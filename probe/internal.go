/**

Per source, we can have the following state transitions



```mermaid # https://mermaid.live/
stateDiagram
	direction LR
	Idle --> Running
	Running --> Failed
	Running --> Idle
	Failed --> Running
	state Running {
		direction LR
		[*] --> Scheduled
		Scheduled --> New
		Scheduled --> Ignored
		New --> Probing
		Probing --> Found
		Probing --> Timeout
		Probing --> Blacklisted
		Timeout --> Reverify
		Reverify --> Scheduled
		Found --> [*]
		Blacklisted --> [*]
	}
````
*/
package probe

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/stats"

	"github.com/rs/zerolog/log"
)

// reverification is kept as state between restarts,
// so it cannot contain any non-serializable state
type reVerify struct {
	Proxy   pmux.Proxy
	Attempt int
	After   int64
}

type internal struct {
	LastReverified   map[pmux.Proxy]reVerify
	Blacklist        map[pmux.Proxy]int
	Seen             map[pmux.Proxy]bool
	SeenSources      map[pmux.Proxy]map[int]bool
	Failures         []string
	ReverifyCounter  int64
	ReverifyAttempts int64

	failuresInverted map[string]int
	scheduled        chan verify
	probing          chan verify
	forget           chan failure
	timeout          chan failure
	stats            *stats.Stats
	found            chan verify
	snapshot         chan chan internal
}

func newInternal(stats *stats.Stats, probing chan verify, buffer int) internal {
	// TODO: eventually add MaxMindDB filter https://pkg.go.dev/github.com/oschwald/geoip2-golang#section-readme
	// and update via https://github.com/maxmind/geoipupdate
	return internal{
		stats:            stats,
		probing:          probing,
		failuresInverted: map[string]int{},
		scheduled:        make(chan verify, buffer),
		forget:           make(chan failure, buffer),
		timeout:          make(chan failure, buffer),
		found:            make(chan verify, buffer),
		snapshot:         make(chan chan internal),
		SeenSources:      make(map[pmux.Proxy]map[int]bool),
		Seen:             make(map[pmux.Proxy]bool),
		Blacklist:        make(map[pmux.Proxy]int),
		LastReverified:   make(map[pmux.Proxy]reVerify),
	}
}

func (i *internal) main(ctx app.Context) {
	var next time.Time
	var delay time.Duration
	for {
		now := time.Now()
		if next.After(now) {
			delay = next.Sub(now)
		}
		start := time.After(delay)
		select {
		case <-ctx.Done():
			return

		case v := <-i.scheduled:
			i.handleScheduled(v)

		case response := <-i.snapshot:
			i.hanldeSnapshot(response)

		case f := <-i.forget:
			i.handleForget(f)
			ctx.Heartbeat()

		case <-start:
			i.handleReverify(ctx.Ctx())
			// TODO: at the moment, two reverifies can be running ...
			sleep := time.Duration(rand.Intn(500))
			next = time.Now().Add(30*time.Minute + sleep*time.Second)

		case f := <-i.timeout:
			i.handleTimeout(f)
			ctx.Heartbeat()

		case v := <-i.found:
			i.handleFound(v)
			ctx.Heartbeat()
		}
	}
}

func (i *internal) handleScheduled(v verify) {
	log := app.Log.From(v.ctx)
	if v.Proxy == 0 {
		i.stats.Update(v.Source, stats.Ignored)
		log.Trace().Msg("empty ip")
		return
	}
	_, ok := i.SeenSources[v.Proxy]
	if ok {
		i.SeenSources[v.Proxy][v.Source] = true
	} else {
		i.SeenSources[v.Proxy] = map[int]bool{v.Source: true}
	}
	_, ok = i.Blacklist[v.Proxy]
	if ok {
		i.stats.Update(v.Source, stats.Ignored)
		delete(i.LastReverified, v.Proxy)
		log.Trace().Msg("was blacklisted")
		return
	}
	_, ok = i.LastReverified[v.Proxy]
	if ok && v.Attempt == 0 {
		i.stats.Update(v.Source, stats.Ignored)
		log.Trace().Msg("in reverify backlog")
		return
	}
	_, ok = i.Seen[v.Proxy]
	if ok {
		i.stats.Update(v.Source, stats.Ignored)
		delete(i.LastReverified, v.Proxy)
		log.Trace().Msg("in pool")
		return
	}
	i.stats.Update(v.Source, stats.New)
	i.probing <- v
}

func (p *internal) requestSnapshot() internal {
	request := make(chan internal)
	defer close(request)
	p.snapshot <- request
	return <-request
}

func (i *internal) hanldeSnapshot(response chan internal) {
	snapshot := internal{
		ReverifyCounter:  i.ReverifyCounter,
		ReverifyAttempts: i.ReverifyAttempts,
		LastReverified:   map[pmux.Proxy]reVerify{},
		Blacklist:        map[pmux.Proxy]int{},
		Seen:             map[pmux.Proxy]bool{},
		SeenSources:      map[pmux.Proxy]map[int]bool{},
		Failures:         make([]string, len(i.Failures)),
	}
	for k, v := range i.LastReverified {
		snapshot.LastReverified[k] = v
	}
	for k, v := range i.Blacklist {
		snapshot.Blacklist[k] = v
	}
	for k, v := range i.Seen {
		snapshot.Seen[k] = v
	}
	for k, v := range i.Failures {
		snapshot.Failures[k] = v
	}
	for k, v := range i.SeenSources {
		snapshot.SeenSources[k] = map[int]bool{}
		for s, t := range v {
			snapshot.SeenSources[k][s] = t
		}
	}
	response <- snapshot
}

const Reverify int = 0

func (i *internal) handleForget(f failure) {
	i.stats.Update(f.v.Source, stats.Blacklisted)
	log := app.Log.From(f.v.ctx)
	delete(i.LastReverified, f.v.Proxy)
	if f.v.Source == Reverify {
		i.ReverifyAttempts += int64(f.v.Attempt)
		i.ReverifyCounter++
	}
	shErr := app.ShErr(f.err) // best-effort in low-cardinality
	idx, ok := i.failuresInverted[shErr.Error()]
	if !ok {
		idx = len(i.Failures)
		i.Failures = append(i.Failures, shErr.Error())
		i.failuresInverted[shErr.Error()] = idx
	}
	i.Blacklist[f.v.Proxy] = idx
	log.Info().Err(shErr).Int("idx", idx).Msg("blacklisted")
}

func (i *internal) handleTimeout(f failure) {
	i.stats.Update(f.v.Source, stats.Timeout)
	i.LastReverified[f.v.Proxy] = reVerify{
		Proxy:   f.v.Proxy,
		Attempt: f.v.Attempt + 1,
		After:   time.Now().Add(1 * time.Hour).Unix(),
	}
	log := app.Log.From(f.v.ctx)
	log.Trace().Msg("verify timeout")
}

var maxReverifies = 5

func (i *internal) handleReverify(ctx context.Context) {
	if i.stats.Snapshot().IsRunning(Reverify) {
		log.Info().Msg("reverify is running")
		return
	}
	now := time.Now().Unix()
	reverify := make(map[pmux.Proxy]reVerify, len(i.LastReverified))
	for k, v := range i.LastReverified {
		if v.Attempt > maxReverifies {
			// only in Go it's allowed to modify hashmap during iteration...
			i.handleForget(failure{
				err: fmt.Errorf("exceeded %d reverifies", maxReverifies),
				v: verify{
					ctx:     ctx,
					Proxy:   v.Proxy,
					Source:  Reverify,
					Attempt: v.Attempt,
				},
			})
			continue
		}
		if v.After > now {
			continue
		}
		reverify[k] = v
	}
	if len(reverify) == 0 {
		return
	}
	log.Info().Int("count", len(reverify)).Msg("reverifying batch")
	go func() {
		ctx := app.Log.WithStr(ctx, "source", "reverify")
		i.stats.LaunchAnticipated(Reverify, len(reverify))
		for _, rv := range reverify {
			ctx := app.Log.WithStringer(ctx, "proxy", rv.Proxy)
			v := verify{ctx, rv.Proxy, Reverify, rv.Attempt}
			i.stats.Update(Reverify, stats.Scheduled)
			select {
			case <-ctx.Done():
				return
			case i.scheduled <- v:
			}
		}
		i.stats.Finish(Reverify, nil)
		log.Info().Msg("reverify batch done")
	}()
}

func (i *internal) handleFound(v verify) {
	if v.Source == Reverify {
		i.ReverifyAttempts += int64(v.Attempt)
		i.ReverifyCounter++
	}
	delete(i.LastReverified, v.Proxy)
	i.Seen[v.Proxy] = true
}
