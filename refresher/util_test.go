package refresher

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/checker"
	"github.com/nfx/slrp/history"
	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/pool"
	"github.com/nfx/slrp/probe"
	"github.com/nfx/slrp/sources"
	"github.com/nfx/slrp/stats"
)

func stubSource() {
	proxy := pmux.Socks4Proxy("127.0.0.1:0")
	refreshDelay = 1 * time.Second
	sources.Sources = []sources.Source{
		{
			ID:        1,
			Frequency: 1 * time.Hour,
			Seed:      true,
		},
		{
			ID:        2,
			Frequency: 1 * time.Hour,
			Feed: func(_ context.Context, _ *http.Client) sources.Src {
				time.Sleep(refreshDelay)
				return proxyArraySrc{proxy}
			},
		},
		{
			ID:        3,
			Frequency: 1 * time.Hour,
			Seed:      true,
			Feed: func(_ context.Context, _ *http.Client) sources.Src {
				time.Sleep(refreshDelay)
				return proxyArraySrc{proxy}
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

func start() (*Refresher, app.MockRuntime, func()) {
	// we might even need mutex here :(
	stubSource()
	singletons := app.Factories{
		"checker":   checker.NewChecker,
		"probe":     probe.NewProbe,
		"stats":     stats.NewStats,
		"pool":      pool.NewPool,
		"refresher": NewRefresher,
		"history":   history.NewHistory,
	}.Init()
	mockRuntime := singletons.MockStart()
	return singletons["refresher"].(*Refresher), mockRuntime, func() {
		mockRuntime.Stop()
	}
}
