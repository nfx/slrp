package pool

import (
	"context"
	"fmt"
	"math/rand"
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
	Entries   []*entry
	incoming  chan incoming
	remove    chan removal
	requests  chan request
	snapshot  chan chan []*entry
	reanimate chan bool
	reply     chan reply
	work      chan work //todo channel in pool
	minute    *time.Ticker
	evictions []pmux.Proxy
	eviction  chan chan []pmux.Proxy
	config    *monitorConfig
}

func (pool *shard) init(config *monitorConfig, work chan work) {
	pool.work = work
	pool.incoming = make(chan incoming)
	pool.remove = make(chan removal)
	pool.requests = make(chan request)
	pool.reanimate = make(chan bool)
	pool.snapshot = make(chan chan []*entry)
	pool.reply = make(chan reply)
	pool.eviction = make(chan chan []pmux.Proxy)
	pool.minute = time.NewTicker(1 * time.Minute)
	pool.config = config
}

func (pool *shard) main(ctx app.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case res := <-pool.snapshot:
			res <- pool.Entries
		case <-pool.minute.C:
			if pool.handleReanimate() {
				ctx.Heartbeat()
			}
			if pool.checkEviction(ctx.Ctx()) {
				ctx.Heartbeat()
			}
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
		case r := <-pool.eviction:
			r <- pool.evictions
			pool.evictions = []pmux.Proxy{}
		}
	}
}

func (pool *shard) checkEviction(ctx context.Context) bool {
	log := app.Log.From(ctx)
	replace := []*entry{}
	evict := []pmux.Proxy{}
	thresholdTimeouts := pool.config.evictThresholdTimeouts
	thresholdFailures := pool.config.evictThresholdFailures
	thresholdReanimations := pool.config.evictThresholdReanimations
	longerSleep := pool.config.longTimeoutSleep
	for i := range pool.Entries {
		e := pool.Entries[i]
		if e.Eviction(thresholdTimeouts, thresholdFailures, thresholdReanimations, longerSleep) {
			proxy := e.Proxy
			log.Info().
				Stringer("proxy", proxy).
				Int("shard", proxy.Bucket(pool.config.shards)).
				Msg("evicting from shard")
			evict = append(evict, proxy)
		} else {
			replace = append(replace, e)
		}
	}
	if len(evict) > 0 {
		pool.Entries = replace
		pool.evictions = append(pool.evictions, evict...)
		return true
	}
	return false
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
	newEntries := []*entry{}
	found := false
	for _, v := range pool.Entries {
		if v.Proxy == r.proxy {
			found = true
			continue
		}
		local := v
		newEntries = append(newEntries, local)
	}
	pool.Entries = newEntries
	if found {
		log := app.Log.From(context.TODO())
		log.Info().Stringer("proxy", r.proxy).Msg("removed")
	}
	r.reply <- found
}

func (pool *shard) add(v incoming) {
	pool.Entries = append(pool.Entries, newEntry(v.Proxy, v.Speed, int16(pool.config.evictSpanMinutes)))
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
	if size == 1 {
		return pool.Entries[0]
	}
	// offset := 0
	// defaultSorting(pool.Entries)

	offset := rand.Intn(size)
	// offset := r.serial % len(pool.Entries)

	// TODO: per-request offset selection strategy -
	// scrapes are encouraged to refresh the old or
	// the least offered proxies, but relays need fresher pool
	available := pool.Entries[offset:size]
	ctx := r.in.Context()
	for idx := range available {
		e := pool.Entries[offset+idx]
		if e.ConsiderSkip(ctx, pool.config.offerLimit) {
			continue
		}
		return e
	}
	return nil
}

func (pool *shard) handleRequest(r request) {
	// log := app.Log.From(r.in.Context())
	r.in.Header.Set("User-Agent", uarand.GetRandom())
	entry := pool.firstAvailableProxy(r)
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
	// TODO: special error codes for timeouts
	if err == nil {
		entry.MarkSuccess()
		// TODO: Bytes1D & Bytes5M
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
	entry.MarkFailure(err, pool.config.shortTimeoutSleep)
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
