package probe

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"net/http"
	"strings"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/checker"
	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/pool"
	"github.com/nfx/slrp/sources"
	"github.com/nfx/slrp/stats"

	_ "github.com/Bogdan-D/go-socks4"
)

type verify struct {
	ctx     context.Context
	Proxy   pmux.Proxy
	Source  int
	Attempt int
}

type failure struct {
	v   verify
	err error
}

type Probe struct {
	pool    *pool.Pool
	stats   *stats.Stats
	checker checker.Checker
	probing chan verify
	state   internal
}

func NewProbe(stats *stats.Stats, p *pool.Pool, c checker.Checker) *Probe {
	buffer := 64 // let's try unbuffered channels to find bugs quicker
	probing := make(chan verify, buffer)
	return &Probe{
		pool:    p,
		checker: c,
		probing: probing,
		stats:   stats,
		state:   newInternal(stats, probing, buffer),
	}
}

func (p *Probe) Schedule(ctx context.Context, proxy pmux.Proxy, source int) bool {
	if proxy == 0 {
		return false
	}
	p.stats.Update(source, stats.Scheduled)
	p.state.scheduled <- verify{ctx, proxy, source, 0}
	return true
}

func (p *Probe) Start(ctx app.Context) {
	go p.state.main(ctx)
	workers := 16
	for w := 0; w < workers; w++ {
		go p.worker(ctx.Ctx())
	}
}

func (p *Probe) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	state := p.state.requestSnapshot()
	gob.NewEncoder(&b).Encode(state)
	return b.Bytes(), nil
}

func (p *Probe) UnmarshalBinary(data []byte) error {
	b := bytes.NewReader(data)
	err := gob.NewDecoder(b).Decode(&p.state)
	if err != nil {
		return err
	}
	// cache inverted failure reason index
	for idx, sherr := range p.state.Failures {
		p.state.failuresInverted[sherr] = idx
	}
	return nil
}

type Stats2 struct {
	Reverify             int
	Blacklist            int
	Seen                 int
	AverageVerifyAttempt int64
	ReverifyAttempts     []int
	Contribution         map[string]int
	Exclusive            map[string]int
	Dirty                map[string]int
}

func (p *Probe) Snapshot() internal {
	return p.state.requestSnapshot()
}

func (p *Probe) HttpDeletetByID(id string, r *http.Request) (interface{}, error) {
	// TODO: harden it
	split := strings.SplitN(id, ":", 3)
	proxy := pmux.NewProxy(fmt.Sprintf("%s:%s", split[1], split[2]), split[0])
	p.state.forget <- failure{
		err: fmt.Errorf("manual remove"),
		v: verify{
			ctx:   r.Context(),
			Proxy: proxy,
		},
	}
	return p.pool.Remove(proxy), nil
}

func (p *Probe) HttpGet(_ *http.Request) (interface{}, error) {
	state := p.state.requestSnapshot()
	attempts := make([]int, maxReverifies+1)
	for _, v := range state.LastReverified {
		attempts[v.Attempt-1]++
	}
	var averageAttempt int64
	if state.ReverifyCounter > 0 {
		averageAttempt = state.ReverifyAttempts / state.ReverifyCounter
	}
	// exclusive items to source
	exclusive := map[string]int{}
	dirty := map[string]int{}
	contribution := map[string]int{}
	names := make([]string, len(sources.Sources)+2) // deleting last source is bad...
	names[0] = "reverify"
	for _, v := range sources.Sources {
		names[v.ID] = v.Name()
	}
	for ip, v := range state.SeenSources {
		for sid := range v {
			_, ok := state.Blacklist[ip]
			if ok {
				continue
			}
			_, ok = state.LastReverified[ip]
			if ok {
				continue
			}
			// dirty is dirty working proxies with dupes
			dirty[names[sid]] += 1
		}
		if len(v) > 1 {
			continue
		}
		for sid := range v {
			// exclusive source contribution
			contribution[names[sid]] += 1
		}
		_, ok := state.Blacklist[ip]
		if ok {
			continue
		}
		_, ok = state.LastReverified[ip]
		if ok {
			continue
		}
		for sid := range v {
			// exclusive source contribution to found working proxies
			exclusive[names[sid]] += 1
		}
	}
	return Stats2{
		Reverify:             len(state.LastReverified),
		Blacklist:            len(state.Blacklist),
		Seen:                 len(state.Seen),
		AverageVerifyAttempt: averageAttempt,
		ReverifyAttempts:     attempts[:],
		Contribution:         contribution,
		Exclusive:            exclusive,
		Dirty:                dirty,
	}, nil
}

func (p *Probe) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case v := <-p.probing:
			p.stats.Update(v.Source, stats.Probing)
			ctx := app.Log.WithStringer(v.ctx, "proxy", v.Proxy)
			speed, err := p.checker.Check(ctx, v.Proxy)
			if err != nil {
				if isTemporary(err) {
					p.state.timeout <- failure{v, err}
				} else {
					p.state.forget <- failure{v, err}
				}
				continue
			}
			p.stats.Update(v.Source, stats.Found)
			p.pool.Add(ctx, v.Proxy, speed)
			p.state.found <- v
		}
	}
}

func isTemporary(err error) bool {
	if err == nil {
		return false
	}
	str := err.Error()
	for _, v := range []string{
		"Maximum number of open connections reached",
		"Too many open connections",
		"Too Many Requests",
		"Gateway Timeout",
		"too many open files",
	} {
		if strings.Contains(str, v) {
			return true
		}
	}
	t, ok := err.(interface {
		Temporary() bool
	})
	return ok && t.Temporary()
}
