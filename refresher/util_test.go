package refresher

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/sources"
	"github.com/nfx/slrp/stats"
)

type nilPool struct{}

func (n nilPool) RandomFast(ctx context.Context) context.Context {
	return ctx
}

type counterProbe map[int]int

func (c counterProbe) Schedule(ctx context.Context, proxy pmux.Proxy, source int) bool {
	c[source] += 1
	return true
}

type mockStats map[int]*stats.Stat

func (m mockStats) Launch(source int) {
	m[source].State = stats.Running
}

func (m mockStats) Finish(source int, err error) {
	if err == nil {
		m[source].State = stats.Idle
		return
	}
	m[source].State = stats.Failed
	m[source].Failure = err.Error()
}

func (m mockStats) Snapshot() stats.Sources {
	s := stats.Sources{}
	for k, v := range m {
		s[k] = *v
	}
	return s
}

func withStats(ref *Refresher) *Refresher {
	m := mockStats{}
	for _, v := range ref.sources() {
		m[v.ID] = &stats.Stat{}
	}
	ref.stats = m
	return ref
}

// var refreshDelay = 1 * time.Second
var stubProxy = pmux.Socks4Proxy("127.0.0.1:0")

var stubSource = []sources.Source{
	{
		ID:        1,
		Frequency: 15 * time.Minute,
		Seed:      true,
	},
	{
		ID:        2,
		Seed:      true,
		Frequency: 1 * time.Hour,
		Feed: func(_ context.Context, _ *http.Client) sources.Src {
			return proxyArraySrc{stubProxy}
		},
	},
	{
		ID:        3,
		Frequency: 1 * time.Hour,
		Seed:      true,
		Session:   true,
		Feed: func(_ context.Context, _ *http.Client) sources.Src {
			return proxyArraySrc{stubProxy}
		},
	},
	{
		ID:        4,
		Frequency: 1 * time.Hour,
		Seed:      true,
		Feed: func(_ context.Context, _ *http.Client) sources.Src {
			return failingSrc("always failing")
		},
	},
	{
		ID:        5,
		Frequency: 1 * time.Second,
		Seed:      true,
		Feed: func(_ context.Context, _ *http.Client) sources.Src {
			return sleepingSrc(300)
		},
	},
}

type proxyArraySrc []pmux.Proxy

func (t proxyArraySrc) Generate(ctx context.Context) <-chan pmux.Proxy {
	out := make(chan pmux.Proxy)
	go func() {
		defer close(out)
		for _, v := range t {
			select {
			case <-ctx.Done():
				return
			case out <- v:
			}
		}
	}()
	return out
}

func (t proxyArraySrc) Err() error {
	return nil
}

func (t proxyArraySrc) Len() int {
	return len(t)
}

type sleepingSrc int

func (t sleepingSrc) Generate(ctx context.Context) <-chan pmux.Proxy {
	out := make(chan pmux.Proxy)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(t) * time.Second):
				return
			}
		}
	}()
	return out
}

func (t sleepingSrc) Err() error {
	return nil
}

func (t sleepingSrc) Len() int {
	return 1
}

type failingSrc string

func (f failingSrc) Generate(ctx context.Context) <-chan pmux.Proxy {
	out := make(chan pmux.Proxy)
	close(out)
	return out
}

func (f failingSrc) Err() error {
	return fmt.Errorf(string(f))
}

func (f failingSrc) Len() int {
	return 100500
}
