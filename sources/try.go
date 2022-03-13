package sources

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pmux"
)

type Src interface {
	Generate(ctx context.Context) <-chan pmux.Proxy
	Err() error
}

func gen(r func() ([]pmux.Proxy, error)) Src {
	return &retriableGenerator{
		out: make(chan pmux.Proxy),
		f:   r,
	}
}

type Feed func(ctx context.Context, h *http.Client) Src

func simpleGen(f func(context.Context, *http.Client) ([]pmux.Proxy, error)) Feed {
	return func(ctx context.Context, h *http.Client) Src {
		return gen(func() ([]pmux.Proxy, error) {
			return f(ctx, h)
		})
	}
}

type retriableGenerator struct {
	out chan pmux.Proxy
	err error
	f   func() ([]pmux.Proxy, error)
	len int
}

func (f *retriableGenerator) Generate(ctx context.Context) <-chan pmux.Proxy {
	go f.generate(ctx)
	return f.out
}

func (f *retriableGenerator) Err() error {
	return f.err // race condition?...
}

func (f *retriableGenerator) Len() int {
	return f.len
}

func skipRetry(format string, args ...interface{}) skip {
	return skip(fmt.Sprintf(format, args...))
}

type skip string

func (se skip) Error() string {
	return string(se)
}

func (f *retriableGenerator) generate(ctx context.Context) {
	defer close(f.out)
	var next time.Time
	var delay time.Duration
	log := app.Log.From(ctx)
	for {
		now := time.Now()
		if next.After(now) {
			delay = next.Sub(now)
		}
		start := time.After(delay)
		select {
		case <-ctx.Done():
			return
		case <-start:
			proxies, err := f.f()
			f.err = err
			if _, ok := err.(skip); ok {
				// we may change the interface to return stream of errors, maybe...
				log.Warn().Err(err).Msg("skipping retry on error")
				return
			}
			if err != nil {
				log.Trace().Err(err).Msg("intermediate source failure")
				sleep := rand.Intn(60)
				next = time.Now().Add(time.Duration(sleep) * time.Second)
				continue
			}
			f.len = len(proxies)
			for _, proxy := range proxies {
				select {
				case <-ctx.Done():
					return
				case f.out <- proxy:
				}
			}
			return
		}
	}
}

type mergeSrc struct {
	srcs []Src
	wg   sync.WaitGroup
	out  chan pmux.Proxy
}

func merged() *mergeSrc {
	return &mergeSrc{
		out: make(chan pmux.Proxy),
	}
}

func (m *mergeSrc) refresh(r func() ([]pmux.Proxy, error)) *mergeSrc {
	m.srcs = append(m.srcs, gen(r))
	return m
}

func (m *mergeSrc) forward(ctx context.Context, src Src) {
	defer m.wg.Done()
	for proxy := range src.Generate(ctx) {
		select {
		case m.out <- proxy:
		case <-ctx.Done():
			return
		}
	}
	log := app.Log.From(ctx)
	log.Debug().Msg("done merged forwarding")
}

func (m *mergeSrc) finish(ctx context.Context) {
	m.wg.Wait()
	close(m.out)
	log := app.Log.From(ctx)
	// TODO: lens
	log.Debug().Int("sources", len(m.srcs)).Msg("done merged source")
}

func (m *mergeSrc) Generate(ctx context.Context) <-chan pmux.Proxy {
	for _, src := range m.srcs {
		m.wg.Add(1)
		go m.forward(ctx, src)
	}
	go m.finish(ctx)
	return m.out
}

func (m *mergeSrc) Err() error {
	for _, src := range m.srcs {
		err := src.Err()
		if err != nil {
			return err
		}
	}
	return nil
}
