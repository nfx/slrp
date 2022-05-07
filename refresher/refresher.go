package refresher

import (
	"context"
	"net/http"
	"os"
	"sort"
	"sync/atomic"
	"time"

	"github.com/nfx/slrp/app"
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
	ctx context.Context
	Err error
}

type plan map[int]time.Duration

type Refresher struct {
	probe    *probe.Probe
	pool     *pool.Pool
	stats    *stats.Stats
	client   *http.Client
	next     atomic.Value
	progress chan progress
	snapshot chan chan plan
	plan     plan
}

func NewRefresher(stats *stats.Stats, pool *pool.Pool, probe *probe.Probe) *Refresher {
	return &Refresher{
		probe:    probe,
		pool:     pool,
		stats:    stats,
		progress: make(chan progress, 1),
		snapshot: make(chan chan plan),
		plan:     plan{},
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
		case f := <-ref.progress:
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
	for _, s := range sources.Sources {
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
	minSourceFrequency := 1 * time.Hour
	for _, v := range sources.Sources {
		if v.Frequency < minSourceFrequency {
			minSourceFrequency = v.Frequency
		}
	}
	nextTrigger := time.Now().Add(minSourceFrequency)
	snapshot := ref.stats.Snapshot()
	// poolSize := ref.pool.Len()
	for _, s := range sources.Sources {
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
			ref.plan[s.ID] = delay
			log.Trace().
				Stringer("wait", delay).
				Msg("still have to wait")
			continue
		}
		log.Trace().Msg("scheduling refresh")
		client := ref.client
		if s.Seed {
			client = http.DefaultClient
		}
		go ref.refresh(sctx, client, s)
	}
	// TODO: maybe bring back nextTrigger someday
	return time.Now().Add(1 * time.Minute)
}

func (ref *Refresher) refresh(ctx context.Context, client *http.Client, source sources.Source) {
	log := app.Log.From(ctx)
	log.Info().Msg("started refresh")
	if source.Session {
		ctx = ref.pool.RandomFast(ctx)
	}
	ref.stats.Launch(source.ID)
	feed := source.Feed(ctx, client)
	for proxy := range feed.Generate(ctx) {
		ctx := app.Log.WithStringer(ctx, "proxy", proxy)
		if !ref.probe.Schedule(ctx, proxy, source.ID) {
			log.Warn().Msg("failed to schedule")
		}

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
	ref.progress <- progress{ctx, feed.Err()}
}
