package refresher

import (
	"context"
	"net/http"
	"os"
	"sort"
	"sync/atomic"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/pool"
	"github.com/nfx/slrp/probe"
	"github.com/nfx/slrp/sources"
	"github.com/nfx/slrp/stats"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.DurationFieldUnit = time.Second
}

type progress struct {
	Source int
	Len    int
}

type finish struct {
	Source int
	ctx    context.Context
	Err    error
}

type status struct {
	Delay   time.Duration
	Started time.Time
	Adds    [15]time.Time
	Added   int
	Len     int
}

func (s *status) Progress() int {
	if s.Len == 0 {
		return 0
	}
	return int(100 * float32(s.Added) / float32(s.Len))
}

func (s *status) EstFinish(queued int) time.Time {
	now := time.Now()
	var measures int
	var maxTime time.Time
	var totalDur time.Duration
	for i := 0; i < len(s.Adds)-1; i++ {
		a := s.Adds[i]
		b := s.Adds[i+1]
		if a.After(b) || a.Equal(b) {
			continue
		}
		if b.After(maxTime) {
			maxTime = b
		}
		totalDur += b.Sub(a)
		measures++
	}
	durPerItem := 0*time.Minute
	if s.Added > 0 && measures > 0 {
		recentDurPerItem := totalDur / time.Duration(measures)
		allDurPerItem := now.Sub(s.Started) / time.Duration(s.Added)
		durPerItem = (recentDurPerItem + allDurPerItem) / 2
	}
	remaining := queued + s.Len - s.Added
	padding := now.Sub(maxTime)
	return now.Add(durPerItem * time.Duration(remaining)).Add(padding)
}

type plan map[int]*status

type Refresher struct {
	probe    probeContract
	pool     poolContract
	stats    statsContract
	client   *http.Client
	next     atomic.Value
	progress chan progress
	finish   chan finish
	snapshot chan chan plan
	sources  func() []sources.Source
	plan     plan
}

type probeContract interface {
	Schedule(ctx context.Context, proxy pmux.Proxy, source int) bool
}

type poolContract interface {
	RandomFast(ctx context.Context) context.Context
}

type statsContract interface {
	Launch(source int)
	Finish(source int, err error)
	Snapshot() stats.Sources
}

func NewRefresher(stats *stats.Stats, pool *pool.Pool, probe *probe.Probe) *Refresher {
	return &Refresher{
		probe:    probe,
		pool:     pool,
		stats:    stats,
		finish:   make(chan finish, 1),
		progress: make(chan progress),
		snapshot: make(chan chan plan),
		plan:     plan{},
		sources: func() []sources.Source {
			return sources.Sources
		},
		client: &http.Client{
			Transport: pool,
		},
	}
}

func (ref *Refresher) Start(ctx app.Context) {
	go ref.main(ctx)
}

func (ref *Refresher) main(ctx app.Context) {
	var delay time.Duration
	for {
		now := time.Now()
		next, ok := ref.next.Load().(time.Time)
		if !ok {
			next = now
		}
		if next.After(now) {
			delay = next.Sub(now)
		}
		start := time.After(delay)
		select {
		case <-ctx.Done():
			return
		case res := <-ref.snapshot:
			snapshot := plan{}
			for k, v := range ref.plan {
				snapshot[k] = v
			}
			res <- snapshot
		case p := <-ref.progress:
			s, ok := ref.plan[p.Source]
			if !ok {
				s = &status{}
				ref.plan[p.Source] = s
			}
			if p.Len == 0 {
				s.Started = time.Now()
				s.Adds = [15]time.Time{}
				s.Added = 0
			}
			s.Len = p.Len
			s.Adds[s.Added%len(s.Adds)] = time.Now()
			s.Added++
			ctx.Heartbeat()
		case f := <-ref.finish:
			s, ok := ref.plan[f.Source]
			if ok {
				s.Added = 0
				s.Len = 0
			}
			log := app.Log.From(f.ctx)
			log.Info().Err(f.Err).Msg("finished refresh")
			ctx.Heartbeat()
		case <-start:
			next = ref.checkSources(ctx.Ctx(), next)
			ref.next.Store(next)
			log.Trace().
				Stringer("next", time.Until(next)).
				Msg("finished checking sources")
			ctx.Heartbeat()
		}
	}
}

func (ref *Refresher) Snapshot() plan {
	out := make(chan plan)
	defer close(out)
	ref.snapshot <- out
	return <-out
}

func (ref *Refresher) HttpGet(_ *http.Request) (interface{}, error) {
	return ref.upcoming(), nil
}

type upcoming struct {
	Source    int
	Delay     time.Duration
	Frequency time.Duration
}

func (ref *Refresher) upcoming() (result []upcoming) {
	snapshot := ref.stats.Snapshot()
	next, ok := ref.next.Load().(time.Time)
	if !ok {
		next = time.Now()
	}
	for _, s := range ref.sources() {
		if s.Feed == nil {
			continue
		}
		v, ok := snapshot[s.ID]
		if !ok {
			result = append(result, upcoming{
				Source:    s.ID,
				Delay:     0,
				Frequency: s.Frequency,
			})
			continue
		}
		if v.State == stats.Running {
			continue
		}
		nextUpdate := v.Updated.Add(s.Frequency)
		if v.State == stats.Failed {
			nextUpdate = next
		}
		until := time.Until(nextUpdate)
		if until < 0 {
			until = 0
		}
		result = append(result, upcoming{
			Source:    s.ID,
			Delay:     until,
			Frequency: s.Frequency,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Delay < result[j].Delay
	})
	return result
}

// package-level variable, that could be overriden in unit tests
var refreshDelay = 1 * time.Minute

func (ref *Refresher) checkSources(ctx context.Context, trigger time.Time) time.Time {
	minSourceFrequency := 60 * time.Minute
	for _, v := range ref.sources() {
		if v.Frequency < minSourceFrequency {
			minSourceFrequency = v.Frequency
		}
	}
	nextTrigger := time.Now().Add(minSourceFrequency)
	snapshot := ref.stats.Snapshot()
	for _, s := range ref.sources() {
		sctx := app.Log.WithStr(ctx, "source", s.Name())
		log := app.Log.From(sctx)
		if s.Feed == nil {
			log.Error().Msg("feed is nil")
			continue
		}
		if snapshot.IsRunning(s.ID) {
			log.Trace().Msg("still refreshing")
			continue
		}
		v, hasStats := snapshot[s.ID]
		nextSourceUpdate := v.Updated.Add(s.Frequency)
		if v.State == stats.Failed {
			// TODO: negative triggers?...
			nextSourceUpdate = trigger.Add(refreshDelay)
		}
		if hasStats && nextSourceUpdate.Before(nextTrigger) {
			nextTrigger = nextSourceUpdate
		}
		if nextSourceUpdate.After(trigger) {
			delay := nextSourceUpdate.Sub(trigger)
			_, ok := ref.plan[s.ID]
			if !ok {
				ref.plan[s.ID] = &status{}
			}
			ref.plan[s.ID].Delay = delay // FIXME: data race?..
			log.Trace().
				Stringer("wait", delay).
				Msg("still have to wait")
			continue
		}
		log.Trace().Msg("scheduling refresh")
		client := ref.client
		if s.Seed {
			// TODO: wrap with history
			client = http.DefaultClient
		}
		go ref.refresh(sctx, client, s)
	}
	// TODO: maybe bring back nextTrigger someday
	return trigger.Add(1 * time.Minute)
}

func (ref *Refresher) refresh(ctx context.Context, client *http.Client, source sources.Source) {
	log := app.Log.From(ctx)
	log.Info().Msg("started refresh")
	if source.Session {
		ctx = ref.pool.RandomFast(ctx)
	}
	ref.stats.Launch(source.ID)
	feed := source.Feed(ctx, client)
	ref.progress <- progress{source.ID, 0}
	for proxy := range feed.Generate(ctx) {
		ctx := app.Log.WithStringer(ctx, "proxy", proxy)
		if !ref.probe.Schedule(ctx, proxy, source.ID) {
			log.Warn().Msg("failed to schedule") // TODO: this happens too often
		}
		ref.progress <- progress{source.ID, feed.Len()}

		// if proxy.Proto == pmux.HTTP {
		// 	if !ref.probe.Schedule(ctx, pmux.Proxy{
		// 		IP:    proxy.IP,
		// 		Port:  proxy.Port,
		// 		Proto: pmux.HTTPS,
		// 	}, source.ID) {
		// 		log.Warn().Msg("failed to schedule")
		// 	}
		// }
		// if proxy.Proto == pmux.HTTPS {
		// 	if !ref.probe.Schedule(ctx, pmux.Proxy{
		// 		IP:    proxy.IP,
		// 		Port:  proxy.Port,
		// 		Proto: pmux.HTTP,
		// 	}, source.ID) {
		// 		log.Warn().Msg("failed to schedule")
		// 	}
		// }
	}
	// TODO: maybe update failed state from a secong goroutine?...
	ref.stats.Finish(source.ID, feed.Err())
	log.Info().Msg("finished refresh")
	ref.finish <- finish{source.ID, ctx, feed.Err()}
}
