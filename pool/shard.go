package pool

import (
	"context"
	"fmt"
	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat/distuv"
	"math"

	"net/http"
	"sort"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pmux"

	"github.com/corpix/uarand"
)

type incoming struct {
	ctx   context.Context
	Proxy pmux.Proxy
	Speed time.Duration
}

type request struct {
	in      *http.Request
	out     chan *http.Response
	start   time.Time
	attempt int
	serial  int
}

type reply struct {
	r        request
	response *http.Response
	start    time.Time
	e        *entry
	err      error
}

type work struct {
	r     request
	e     *entry
	reply chan reply
}

type removal struct {
	proxy pmux.Proxy
	reply chan bool
}

type shard struct {
	Entries   []entry
	incoming  chan incoming
	remove    chan removal
	requests  chan request
	snapshot  chan chan []entry
	reanimate chan bool
	reply     chan reply
	work      chan work //todo channel in pool
	minute    *time.Ticker
	ticks     int
	src       *rand.Rand
}

func (pool *shard) init(work chan work) {
	pool.work = work
	pool.incoming = make(chan incoming)
	pool.remove = make(chan removal)
	pool.requests = make(chan request)
	pool.reanimate = make(chan bool)
	pool.snapshot = make(chan chan []entry)
	pool.reply = make(chan reply)
	pool.minute = time.NewTicker(1 * time.Minute)
	pool.src = rand.New(rand.NewSource(uint64(now().Unix())))
}

func (pool *shard) main(ctx app.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case res := <-pool.snapshot:
			snapshot := make([]entry, len(pool.Entries))
			for i := range pool.Entries {
				snapshot[i] = pool.Entries[i]
			}
			res <- snapshot
		case <-pool.minute.C:
			if pool.handleReanimate() {
				ctx.Heartbeat()
			}
			nt := float64(0.0)

			for i := 0; i < 100; i++ {
				e := pool.ThompsonBandit()
				nt += float64(e.SuccessRate())

			}
			nt = nt / float64(100)
			fmt.Printf("\n\n\n\nAVERAGEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE  %.5f\n\n\n\n", nt)
		case <-pool.reanimate:
			pool.forceReanimate()
		case v := <-pool.remove:
			pool.removeProxy(v)
			ctx.Heartbeat()
		case v := <-pool.incoming:
			pool.add(v)
			ctx.Heartbeat()
		case r := <-pool.requests:
			pool.handleRequest(r)
		case r := <-pool.reply:
			pool.handleReply(r)
		}
	}
}

func (pool *shard) handleReanimate() bool {
	heartbeat := false
	for i := range pool.Entries {
		if pool.Entries[i].ReanimateIfNeeded() {
			heartbeat = true
		}
	}
	return heartbeat
}

func (pool *shard) forceReanimate() {
	for i := range pool.Entries {
		pool.Entries[i].ForceReanimate()
	}
}

func (pool *shard) removeProxy(r removal) {
	newEntries := []entry{}
	found := false
	for _, v := range pool.Entries {
		if v.Proxy == r.proxy {
			found = true
			continue
		}
		newEntries = append(newEntries, v)
	}
	pool.Entries = newEntries
	if found {
		log := app.Log.From(context.TODO())
		log.Info().Stringer("proxy", r.proxy).Msg("removed")
	}
	r.reply <- found
}

func (pool *shard) add(v incoming) {
	now := time.Now()
	pool.Entries = append(pool.Entries, entry{
		Proxy:     v.Proxy,
		FirstSeen: now.Unix(),
		LastSeen:  now.Unix(),
		Speed:     v.Speed,
		Seen:      1,
		Offered:   1,
		Ok:        true,
	})
	sort.Slice(pool.Entries, func(i, j int) bool {
		return pool.Entries[i].Speed < pool.Entries[j].Speed
	})
	log := app.Log.From(v.ctx)
	log.Info().Stringer("proxy", v.Proxy).Dur("speed", v.Speed).Msg("added")
}

func (pool *shard) firstAvailableProxy(r request) *entry {
	size := len(pool.Entries)
	if size == 0 {
		return nil
	}
	// offset := 0
	// defaultSorting(pool.Entries)

	offset := rand.Intn(size)
	// offset := r.serial % len(pool.Entries)

	// TODO: per-request offset selection strategy -
	// scrapes are encouraged to refresh the old or
	// the least offered proxies, but relays need fresher pool
	available := pool.Entries[offset:size]
	for idx := range available {
		e := &pool.Entries[offset+idx]
		if e.ConsiderSkip(r.in.Context()) {
			continue
		}
		return e
	}
	return nil
}

func (pool *shard) BanditSuccessRateProxy() *entry {

	//ctx = app.Log.WithStringer(ctx, "proxy", entry.Proxy)
	size := len(pool.Entries)
	if size == 0 {
		return nil
	}
	if size == 1 {
		return &pool.Entries[0]
	}
	//inspired by https://cybernetist.com/2019/01/24/random-weighted-draws-in-go/
	cdf := make([]float64, size)
	weights := make([]float64, size)
	const explorationRate = 0.04
	for idx, e := range pool.Entries {
		weights[idx] = float64(e.SuccessRate())
	}
	floats.CumSum(cdf, weights)
	val := distuv.UnitUniform.Rand() * cdf[len(cdf)-1]
	bandi := sort.Search(len(cdf), func(i int) bool { return cdf[i] > val })

	//fmt.Printf("pool %v (%v/%v)\n", &pool, bandi, size)
	if bandi == size {
		bandi -= 1
	}
	return &pool.Entries[bandi]
}

func (pool *shard) BanditUCBProxy() *entry {

	//ctx = app.Log.WithStringer(ctx, "proxy", entry.Proxy)
	size := len(pool.Entries)
	if size == 0 {
		return nil
	}
	if size == 1 {
		return &pool.Entries[0]
	}
	//inspired by https://cybernetist.com/2019/01/24/random-weighted-draws-in-go/
	//cdf := make([]float64, size)
	weights := make([]float64, size)
	const explorationRate = 0.005
	bandi := 0

	nt := 1
	for _, e := range pool.Entries {
		nt += e.actions
	}
	explore := false
	sucesses := 0
	maxv := float64(-1)
	if rand.Float64() < 0.01 {
		bandi = rand.Intn(size)
		explore = true
	} else {
		for idx, e := range pool.Entries {
			sq := math.Sqrt(math.Log(float64(nt))/float64(e.actions+1)) * explorationRate
			qa := float64(e.SuccessRate() / 100)
			wall := sq + qa //+ rand.Float64()*0.0000001
			sucesses += e.Succeed
			weights[idx] = wall
			if wall > maxv {
				maxv = wall
				bandi = idx
			}
		}
	}
	//cdf := make([]float64, size)
	//floats.CumSum(cdf, weights)
	//val := distuv.UnitUniform.Rand() * cdf[len(cdf)-1]
	//bandi = sort.Search(len(cdf), func(i int) bool { return cdf[i] > val })

	if bandi == size {
		bandi -= 1
	}
	b := pool.Entries[bandi]
	fmt.Printf("pool %v %v %.3f \t %v \t(%v/%v) \t%.3f  sel: %v of: %v, suc:%v \t suc/nt%.4f\n", &pool, b.Proxy.IP(), b.SuccessRate(), explore, bandi, size, weights[bandi], b.actions, b.Offered, b.Succeed, float64(b.actions)/float64(nt))

	pool.Entries[bandi].actions += 1
	return &pool.Entries[bandi]
}
func (pool *shard) ThompsonBandit() *entry {

	//ctx = app.Log.WithStringer(ctx, "proxy", entry.Proxy)
	size := len(pool.Entries)
	if size == 0 {
		return nil
	}
	if size == 1 {
		return &pool.Entries[0]
	}
	//inspired by https://cybernetist.com/2019/01/24/random-weighted-draws-in-go/
	//cdf := make([]float64, size)
	weights := make([]float64, size)
	const explorationRate = 50
	bandi := 0

	nt := float64(0.1)
	sucesses := 0
	for _, e := range pool.Entries {
		nt += float64(e.SuccessRate())

	}
	//explore := false

	mv := float64(-1)

	for idx, e := range pool.Entries {
		//sq := math.Sqrt(math.Log(float64(nt))/float64(e.actions+1)) * explorationRate
		//qa := float64(e.SuccessRate() / 100)
		//wall := sq + qa //+ rand.Float64()*0.0000001
		//sucesses += e.Succeed
		//if e.ConsiderSkip(r.in.Context()) {
		//	continue
		//}
		e.Offered += 1
		a := e.Offered
		sucesses += a
		b := e.Offered - e.Succeed

		bet := distuv.Beta{Alpha: float64(a + 1), Beta: float64(b + 1), Src: pool.src}
		weights[idx] = bet.Rand()
		if weights[idx] > mv {
			bandi = idx
			mv = weights[idx]
		}
	}

	//cdf := make([]float64, size)
	//floats.CumSum(cdf, weights)
	//val := distuv.UnitUniform.Rand() * cdf[len(cdf)-1]
	//bandi = sort.Search(len(cdf), func(i int) bool { return cdf[i] > val })

	if bandi == size {
		bandi -= 1
	}
	b := &pool.Entries[bandi]
	b.actions += 1
	fmt.Printf("pool %v %v \t(%v/%v) \t%v \t %v \t%v \t%.4f\t SR:%.4f\n", &pool, b.Proxy.IP(), bandi, size, weights[bandi], b.Offered, b.Succeed, float64(b.SuccessRate()/100), float64(nt)/float64(size))

	return b
}

func (pool *shard) handleRequest(r request) {
	// log := app.Log.From(r.in.Context())
	r.in.Header.Set("User-Agent", uarand.GetRandom())

	var entry *entry
	if pool.ticks < 10000 {
		entry = pool.firstAvailableProxy(r)

	} else {
		//for i := 0; i < 100; i++ {
		entry = pool.ThompsonBandit()

		//	if entry != nil && entry.ReanimateAfter.After(now()) {
		//		continue
		//	}
		//	break
		//}

	}
	pool.ticks += 1
	if entry == nil {
		// this pool has no entries, try next one
		headers := http.Header{}
		headers.Add("X-Proxy-Serial", fmt.Sprintf("%d", r.serial))
		r.out <- &http.Response{
			StatusCode: 552,
			Status:     "Proxy Pool Exhausted",
			Header:     headers,
			Request:    r.in,
		}
		return
	}
	entry.actions += 1
	// log.Debug().
	// 	Stringer("url", req.URL).
	// 	Stringer("proxy", entry.Proxy).
	// 	Dur("t", time.Since(start)).
	// 	Msg("prepare request")
	ctx := entry.Proxy.InContext(r.in.Context())
	ctx = app.Log.WithStringer(ctx, "proxy", entry.Proxy)
	ctx = app.Log.WithStr(ctx, "serial", fmt.Sprint(r.serial))
	r.in = r.in.WithContext(ctx)
	go func() {
		// send work via goroutine, so we're not deadlocked by work trying to send to handleReply
		// this leaks a goroutine to certain extent, but hopefully with throttling on serial channel
		// we can overcome this difficulty
		select {
		case <-ctx.Done():
			return
		case pool.work <- work{
			reply: pool.reply,
			e:     entry,
			r:     r,
		}: // bada boom ts
		}
	}()
}

func (pool *shard) handleReply(r reply) {
	request := r.r
	res := r.response
	log := app.Log.From(r.r.in.Context())
	err := r.err
	entry := r.e
	if err == nil && res.StatusCode >= 400 {
		err = fmt.Errorf(res.Status)
	}
	if err == nil {
		entry.MarkSuccess()
		res.Header.Set("X-Proxy-Through", entry.Proxy.String())
		res.Header.Set("X-Proxy-Attempt", fmt.Sprint(request.attempt))
		res.Header.Set("X-Proxy-Offered", fmt.Sprint(entry.Offered))
		res.Header.Set("X-Proxy-Succeed", fmt.Sprint(entry.Succeed))
		res.Header.Set("X-Proxy-Serial", fmt.Sprint(request.serial))
		log.Debug().
			Stringer("t", time.Since(request.start)).
			Int("offered", entry.Offered).
			Msg("forwarded")
		r.r.out <- res
		return
	}
	// TODO: if more than 10 failed offers in this hour, mark dead till beginning of next hour
	entry.MarkFailure(err)
	log.Debug().
		Err(app.ShErr(err)).
		Int("timeouts", entry.Timeouts).
		Int("offered", entry.Offered).
		Stringer("t", time.Since(request.start)).
		Msg("forwarding failed")
	if request.attempt >= 10 {
		headers := http.Header{}
		headers.Add("X-Proxy-Serial", fmt.Sprintf("%d", request.serial))
		request.out <- &http.Response{
			StatusCode: 429,
			Status:     err.Error(),
			Header:     headers,
			Request:    request.in,
		}
		return
	}
	// todo: proxies becoming dead (connections rejected, etc)
	request.out <- nil
}
