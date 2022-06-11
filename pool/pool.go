package pool

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/history"
	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/ql"

	_ "github.com/Bogdan-D/go-socks4"
)

type Pool struct {
	work          chan work
	serial        chan int
	pressure      chan int
	halt          chan int
	client        *http.Client
	shards        []shard
	workerCancels []context.CancelFunc
}

func NewPool(history *history.History) *Pool {
	return &Pool{
		serial:   make(chan int),
		work:     make(chan work, 128),
		pressure: make(chan int),
		halt:     make(chan int),
		shards:   make([]shard, 32),
		client: &http.Client{
			Transport: history.Wrap(pmux.ContextualHttpTransport()),
			Timeout:   10 * time.Second,
		},
	}
}

func (pool *Pool) Start(ctx app.Context) {
	go pool.counter(ctx)
	go pool.halter(ctx)
	for i := range pool.shards {
		shard := &pool.shards[i]
		shard.init(pool.work)
		go shard.main(ctx)
		shard.reanimate <- true
	}
	parallelRequests := cap(pool.work)
	for i := 0; i < parallelRequests; i++ {
		ctx, cancel := context.WithCancel(ctx.Ctx())
		pool.workerCancels = append(pool.workerCancels, cancel)
		go pool.worker(ctx)
	}
}

func (pool *Pool) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case w := <-pool.work:
			start := time.Now()
			res, err := pool.client.Do(w.r.in)
			w.reply <- reply{
				start:    start,
				response: res,
				err:      err,
				e:        w.e,
				r:        w.r,
			}
		}
	}
}

func (pool *Pool) halter(ctx app.Context) {
	var pressure int
	log := app.Log.From(ctx.Ctx())
	for {
		// leaky bucket backpressure
		if pressure > 32 {
			select {
			case <-ctx.Done():
				return
			case pool.halt <- 1:
				pressure = 0
				log.Warn().
					Int("pressure", pressure).
					Msg("too many errors, slowing down")
			}
		}
		select {
		case <-ctx.Done():
			return
		case s := <-pool.pressure:
			log.Warn().Int("serial", s).Msg("pressure received")
			pressure++
		}
	}
}

// embarassingly simple implementation of lockless counter.
// Initially thought to be implemented with atomic.AddUint32,
// though results are consistent only within one CPU core, as
// value is not propagated to L1 caches of all cores and this
// is not Java. This is also the place that implements
// backpressure via leaky bucket.
func (pool *Pool) counter(ctx app.Context) {
	var serial int
	var delay time.Duration
	var start <-chan time.Time
	var delayed bool
	log := app.Log.From(ctx.Ctx())
	// halfFull := int(0.5 * float32(cap(pool.work)))
	for {
		// if len(pool.work) >= halfFull {
		// 	log.Warn().Msg("queue more than half full, slowing a bit")
		// 	delay = 1 * time.Second
		// }
		start = time.After(delay)
		select {
		case <-ctx.Done():
			return
		case <-pool.halt:
			delayed = true
			delay = 1 * time.Minute
			log.Warn().Stringer("delay", delay).Msg("slowing down")
		case <-start:
			serial++
			pool.serial <- serial
			if delayed {
				// otherwise we'll have one request per minute
				delay = 0
			}
		}
	}
}

func (pool *Pool) snapshot() (entries []entry) {
	// https://github.com/orcaman/concurrent-map/blob/893feb299719d9cbb2cfbe08b6dd4eb567d8039d/concurrent_map.go#L161-L240
	var wg sync.WaitGroup
	bc := make(chan []entry)
	for i := range pool.shards {
		sc := pool.shards[i].snapshot
		ch := make(chan []entry, 1)
		wg.Add(1)
		go func() { // todo: add context.Done
			sc <- ch
			entries := <-ch
			bc <- entries
			close(ch)
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(bc)
	}()
	for e := range bc {
		entries = append(entries, e...)
	}
	defaultSorting(entries)
	return
}

// TODO: think of rather type Facet struct { Name, Field string; Values []string }
type Card struct {
	Name  string
	Value interface{}
}

type PoolStats struct {
	Total   int
	Cards   []Card
	Entries []entry
}

func (pool *Pool) HttpGet(r *http.Request) (interface{}, error) {
	filter := r.FormValue("filter")
	if filter == "" {
		filter = "Offered > 1 ORDER BY LastSeen DESC"
	}
	result := PoolStats{}
	snapshot := pool.snapshot()
	err := ql.Execute(&snapshot, &result.Entries, filter, func(all *[]entry) {
		var http, https, socks4, socks5, alive, offered, succeeded int
		result.Total = len(*all)
		for _, v := range *all {
			switch v.Proxy.Proto() {
			case pmux.HTTP:
				http++
			case pmux.HTTPS:
				https++
			case pmux.SOCKS4:
				socks4++
			case pmux.SOCKS5:
				socks5++
			}
			if v.Ok {
				alive++
			}
			offered += v.Offered
			succeeded += v.Succeed
		}
		result.Cards = []Card{
			{"Alive", alive},
			{"Success rate", float32(succeeded) / float32(offered)},
			{"HTTP proxies", http},
			{"HTTPS proxies", https},
			{"SOCKS4 proxies", socks4},
			{"SOCKS5 proxies", socks5},
		}
	}, ql.DefaultOrder{ql.Desc("LastSeen")}, ql.DefaultLimit(20))
	return result, err
}

func (pool *Pool) Len() (res int) {
	return len(pool.snapshot())
}

func (pool *Pool) Add(ctx context.Context, proxy pmux.Proxy, speed time.Duration) {
	shard := proxy.Bucket(len(pool.shards))
	pool.shards[shard].incoming <- incoming{ctx, proxy, speed}
}

func (pool *Pool) Remove(proxy pmux.Proxy) bool {
	out := make(chan bool)
	defer close(out)
	shard := proxy.Bucket(len(pool.shards))
	pool.shards[shard].remove <- removal{proxy, out}
	return <-out
}

func (pool *Pool) RandomFast(ctx context.Context) context.Context {
	snapshot := []entry{}
	for _, e := range pool.snapshot() {
		if e.Speed > 1*time.Second {
			continue
		}
		snapshot = append(snapshot, e)
	}
	offset := rand.Intn(len(snapshot))
	return snapshot[offset].Proxy.InContext(ctx)
}

// Session rotates a random proxy per entire fn(ctx, client) call
func (pool *Pool) Session(ctx context.Context, fn func(context.Context, *http.Client) error) error {
	snapshot := []entry{}
	// make a copy from very fast ones, otherwise too complicated for now...
	for _, e := range pool.snapshot() {
		if e.Speed > 1*time.Second {
			continue
		}
		snapshot = append(snapshot, e)
	}
	var attempts int
	for {
		offset := rand.Intn(len(snapshot))
		ctx := snapshot[offset].Proxy.InContext(ctx)
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			attempts++
			err := fn(ctx, pool.client)
			if err != nil && attempts < 10 {
				log := app.Log.From(ctx)
				log.Trace().Err(err).Msg("retrying")
				continue
			}
			return err
		}
	}
}

var ErrNoProxiesLeft = fmt.Errorf("no proxies left")

func (pool *Pool) nextSerial(ctx context.Context) int {
	start := time.Now()
	serial := <-pool.serial
	log := app.Log.From(ctx)
	log.Debug().Stringer("t", time.Since(start)).Msg("nextSerial")
	return serial
}

func (pool *Pool) RoundTrip(req *http.Request) (res *http.Response, err error) {
	// get sequence number and do some throttling if needed
	ctx := req.Context()
	start := time.Now()
	serial := pool.nextSerial(ctx)
	// add trace information deep to all other places
	ctx = app.Log.WithInt(ctx, "serial", serial)
	req = req.WithContext(ctx)
	attempt := 0
	log := app.Log.From(ctx)
	for {
		attempt++
		log := log.With().Int("attempt", attempt).Logger()
		if ctx.Err() != nil {
			log.Info().
				Err(ctx.Err()).
				Dur("t", time.Since(start)).
				Msg("original request failed")
			return nil, ctx.Err()
		}
		out := make(chan *http.Response)
		// shard := rand.Intn(len(pool.shards))
		shard := (serial + attempt) % len(pool.shards)
		log.Trace().Int("shard", shard).Msg("try")
		// set attempt and serial for history wrapper to pick up
		req.Header.Set("X-Proxy-Serial", fmt.Sprint(serial))
		req.Header.Set("X-Proxy-Attempt", fmt.Sprint(attempt))
		// send over the request to one of the shards for randomization purposes
		pool.shards[shard].requests <- request{
			in:      req,
			out:     out,
			start:   start,
			serial:  serial,
			attempt: attempt,
		}
		res := <-out
		// when no response is returned or proxy pool is exhausted
		if attempt < len(pool.shards) && res.StatusCode == 552 {
			continue
		}
		if res == nil {
			continue
		}
		// if res.StatusCode == 552 && pool.pressure != nil {
		// 	// this could mean either no proxies left or all attempts exhausted
		// 	s := time.Now()
		// 	log.Warn().Msg("sending pressure")
		// 	pool.pressure <- serial // livelock....
		// 	log.Warn().Stringer("t", time.Since(s)).Msg("sent pressure")
		// }
		return res, nil
	}
}

func (pool *Pool) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	snapshot := pool.snapshot()
	gob.NewEncoder(&b).Encode(snapshot)
	return b.Bytes(), nil
}

func (pool *Pool) UnmarshalBinary(data []byte) error {
	b := bytes.NewReader(data)
	var snapshot []entry
	err := gob.NewDecoder(b).Decode(&snapshot)
	if err != nil {
		return err
	}
	for _, v := range snapshot {
		shard := v.Proxy.Bucket(len(pool.shards))
		pool.shards[shard].Entries = append(pool.shards[shard].Entries, v)
	}
	return nil
}
