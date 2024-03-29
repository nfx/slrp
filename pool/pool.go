package pool

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/history"
	"github.com/nfx/slrp/ipinfo"
	"github.com/nfx/slrp/pmux"

	_ "github.com/bdandy/go-socks4"
)

type Pool struct {
	ipLookup        ipinfo.IpInfoGetter
	work            chan work
	serial          chan int
	pressure        chan int
	halt            chan time.Duration
	client          httpClient
	shards          []shard
	workerCancels   []context.CancelFunc
	workerProgress  chan int
	pendingEviction []pmux.Proxy // TODO: keep in state
	eviction        chan chan []pmux.Proxy
	minute          *time.Ticker
	config          *monitorConfig
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

func NewPool(history *history.History, ipLookup ipinfo.IpInfoGetter, dialer dialer) *Pool {
	return &Pool{
		ipLookup:       ipLookup,
		serial:         make(chan int),
		pressure:       make(chan int),
		halt:           make(chan time.Duration),
		minute:         time.NewTicker(1 * time.Minute),
		eviction:       make(chan chan []pmux.Proxy),
		workerProgress: make(chan int),
		client: &http.Client{
			Transport: history.Wrap(&http.Transport{
				DialContext:     dialer.DialContext,
				Proxy:           pmux.ProxyFromContext,
				TLSClientConfig: pmux.DefaultTlsConfig,
			}),
		},
	}
}

type monitorConfig struct {
	shards                     int           // 31
	offerLimit                 int           // 25
	evictSpanMinutes           int           // 5
	shortTimeoutSleep          time.Duration // 1m
	longTimeoutSleep           time.Duration // 1h
	evictThresholdTimeouts     int           // 3
	evictThresholdFailures     int           // 3
	evictThresholdReanimations int           // 10
}

func (pool *Pool) Configure(c app.Config) error {
	poolWorkSize := c.IntOr("request_workers", 512)
	pool.work = make(chan work, poolWorkSize)

	if hc, ok := pool.client.(*http.Client); ok {
		requestTimeout := c.DurOr("request_timeout", 10*time.Second)
		hc.Timeout = requestTimeout
	}

	// see https://github.com/nfx/slrp/issues/130
	poolShards := c.IntOr("shards", 1) // 31
	pool.shards = make([]shard, poolShards)

	// evict_span_minutes is a config, that is not compatible with previous state snapshot:
	// count circular buffers will become of a different size and would cause comparison
	// errors in certain edge cases. to prevent this, we can store the config as data
	// and alert early.
	pool.config = &monitorConfig{
		shards:                     poolShards,
		offerLimit:                 c.IntOr("offer_limit", 25),
		evictSpanMinutes:           c.IntOr("evict_span_minutes", 5),
		shortTimeoutSleep:          c.DurOr("short_timeout_sleep", 1*time.Minute),
		longTimeoutSleep:           c.DurOr("long_timeout_sleep", 1*time.Hour),
		evictThresholdTimeouts:     c.IntOr("evict_threshold_timeouts", 3),
		evictThresholdFailures:     c.IntOr("evict_threshold_failures", 3),
		evictThresholdReanimations: c.IntOr("evict_threshold_reanimations", 10),
	}

	return nil
}

func (pool *Pool) Start(ctx app.Context) {
	if pool.config == nil {
		panic("pool is not configured")
	}
	go pool.counter(ctx)
	go pool.halter(ctx)
	go pool.gatherEvictions(ctx)
	for i := range pool.shards {
		shard := &pool.shards[i]
		shard.init(pool.config, pool.work)
		go shard.main(ctx)
		shard.reanimate <- true
	}
	parallelRequests := cap(pool.work)
	go pool.workerMonitor(ctx.Ctx())
	for i := 0; i < parallelRequests; i++ {
		ctx, cancel := context.WithCancel(ctx.Ctx())
		pool.workerCancels = append(pool.workerCancels, cancel)
		go pool.worker(ctx)
	}
}

func (pool *Pool) PendingEviction() []pmux.Proxy {
	req := make(chan []pmux.Proxy, 1)
	defer close(req)
	pool.eviction <- req
	return <-req
}

func (pool *Pool) gatherEvictions(ctx app.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-pool.minute.C:
			var wg sync.WaitGroup
			gather := make(chan []pmux.Proxy)
			for i := range pool.shards {
				shardChannel := pool.shards[i].eviction
				wg.Add(1)
				go func() { // todo: add context.Done
					reply := make(chan []pmux.Proxy)
					defer close(reply)
					shardChannel <- reply
					gather <- <-reply
					wg.Done()
				}()
			}
			go func() {
				wg.Wait()
				close(gather)
			}()
			log := app.Log.From(ctx.Ctx())
			for batch := range gather {
				for _, proxy := range batch {
					log.Info().Stringer("proxy", proxy).Msg("scheduling from pool eviction")
					pool.pendingEviction = append(pool.pendingEviction, proxy)
				}
			}
			pendingCount := len(pool.pendingEviction)
			if pendingCount > 0 {
				log.Info().Int("count", pendingCount).Msg("pending pool eviction")
			}
		case r := <-pool.eviction:
			r <- pool.pendingEviction
			pool.pendingEviction = []pmux.Proxy{}
		}
	}
}

func (pool *Pool) workerMonitor(ctx context.Context) {
	var requests int
	ticker := time.NewTicker(15 * time.Second)
	log := app.Log.From(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case i := <-pool.workerProgress:
			requests += i
		case <-ticker.C:
			log.Info().Int("requests", requests).Msg("active requests")
		}
	}
}

func (pool *Pool) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case w := <-pool.work:
			start := time.Now()
			pool.workerProgress <- 1
			res, err := pool.client.Do(w.r.in)
			pool.workerProgress <- -1
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
	// TODO: make configurable
	slowDown := time.Minute * 1
	maxPressure := 32
	log := app.Log.From(ctx.Ctx())
	for {
		// leaky bucket backpressure
		if pressure > maxPressure {
			select {
			case <-ctx.Done():
				return
			case pool.halt <- slowDown:
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
		case delay = <-pool.halt:
			delayed = true
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

func (pool *Pool) snapshot() (entries []*entry) {
	// https://github.com/orcaman/concurrent-map/blob/893feb299719d9cbb2cfbe08b6dd4eb567d8039d/concurrent_map.go#L161-L240
	var wg sync.WaitGroup
	bc := make(chan []*entry)
	for i := range pool.shards {
		sc := pool.shards[i].snapshot
		ch := make(chan []*entry, 1)
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
	Value any
}

type PoolStats struct {
	Total   int
	Cards   []Card
	Entries []ApiEntry
}

//go:generate go run ../ql/generator/main.go ApiEntry
type ApiEntry struct {
	Proxy          pmux.Proxy `facet:"Protocol"`
	FirstSeen      int64
	LastSeen       int64
	ReanimateAfter time.Time
	Ok             bool
	Speed          time.Duration
	Timeouts       int
	Offered        int
	Reanimated     int
	Succeed        int
	HourOffered    [24]int
	HourSucceed    [24]int
	Country        string `facet:"Country"`
	Provider       string
	ASN            uint16
}

func (d ApiEntryDataset) getProxyProtocol(record int) string {
	return d[record].Proxy.Scheme()
}

func (pool *Pool) HttpGet(r *http.Request) (any, error) {
	filter := r.FormValue("filter")
	if filter == "" {
		filter = "Ok ORDER BY LastSeen DESC"
	}
	var tmp ApiEntryDataset
	for _, v := range pool.snapshot() {
		info := pool.ipLookup.Get(v.Proxy)
		tmp = append(tmp, ApiEntry{
			Proxy:          v.Proxy,
			FirstSeen:      v.FirstSeen,
			LastSeen:       v.LastSeen,
			ReanimateAfter: v.ReanimateAfter,
			Ok:             v.Ok,
			Speed:          v.Speed,
			Timeouts:       v.TimeoutShort.Sum(),
			Offered:        v.RequestsShort(), // TODO: make sure the same time interval
			Reanimated:     v.Reanimated,
			Succeed:        v.SuccessShort.Sum(),
			HourOffered:    v.HourOffered,
			HourSucceed:    v.HourSucceed,
			Country:        info.Country,
			Provider:       info.Provider,
			ASN:            info.ASN,
		})
	}
	return tmp.Query(filter)
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
	snapshot := []*entry{}
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
func (pool *Pool) Session(ctx context.Context, fn func(context.Context, httpClient) error) error {
	snapshot := []*entry{}
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
		select {
		case <-ctx.Done():
			// todo: tune error message so that it's less harsh
			log.Info().
				Err(ctx.Err()).
				Dur("t", time.Since(start)).
				Msg("original request failed")
			return nil, ctx.Err()
		default:
			out := make(chan *http.Response)
			// shard := rand.Intn(len(pool.shards))
			// shart from the first shard to reduce the number of test attempts
			shard := (serial - 1 + attempt - 1) % len(pool.shards)
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
			if res == nil {
				continue
			}
			// when no response is returned or proxy pool is exhausted
			if attempt < len(pool.shards) && res.StatusCode == 552 {
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
}

func (pool *Pool) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	snapshot := pool.snapshot()
	gob.NewEncoder(&b).Encode(snapshot)
	return b.Bytes(), nil
}

func (pool *Pool) UnmarshalBinary(data []byte) error {
	b := bytes.NewReader(data)
	var snapshot []*entry
	err := gob.NewDecoder(b).Decode(&snapshot)
	if err != nil {
		return err
	}
	for _, v := range snapshot {
		local := v
		shard := local.Proxy.Bucket(len(pool.shards))
		pool.shards[shard].Entries = append(pool.shards[shard].Entries, local)
	}
	return nil
}
