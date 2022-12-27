package sources

import (
	"context"
	"errors"
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
	Len() int
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
	return f.len // race condition?...
}

func (f *retriableGenerator) generate(ctx context.Context) {
	defer close(f.out)
	var next time.Time
	var delay time.Duration
	log := app.Log.From(ctx)
	defer log.Debug().Msg("done simple forwarding")
	defer func() {
		// in case something really unforeseen happens within the source
		p := recover()
		if p != nil {
			log.Error().Msgf("panic: %v", p)
		}
	}()
	for {
		now := time.Now()
		if next.After(now) {
			delay = next.Sub(now)
		}
		start := time.After(delay)
		select {
		case <-ctx.Done():
			log.Trace().Msg("stopped trying to forward")
			return
		case <-start:
			proxies, err := f.f()
			f.err = err
			if se, ok := err.(sourceError); ok {
				// contextualize errors
				evt := log.Debug().Err(errors.New(se.msg))
				for _, f := range se.fields {
					f.Apply(evt)
				}
				if se.skip {
					evt.Msg("skipping retry")
					return
				}
				evt.Msg("intermediate failure")
				sleep := rand.Intn(15)
				next = time.Now().Add(time.Duration(sleep) * time.Second)
				continue
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
					log.Trace().Msg("stopped forwarding")
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
	log := app.Log.From(ctx)
	for proxy := range src.Generate(ctx) {
		select {
		case m.out <- proxy:
		case <-ctx.Done():
			log.Trace().Msg("stopped merge forward")
			return
		}
	}
	log.Debug().Msg("done merged forwarding")
}

func (m *mergeSrc) finish(ctx context.Context) {
	m.wg.Wait()
	log := app.Log.From(ctx)
	log.Debug().Int("sources", len(m.srcs)).Msg("done merged source")
	close(m.out)
}

func (m *mergeSrc) Generate(ctx context.Context) <-chan pmux.Proxy {
	for _, src := range m.srcs {
		m.wg.Add(1)
		go m.forward(ctx, src)
	}
	go m.finish(ctx)
	return m.out
}

func (m *mergeSrc) Len() int {
	items := 0
	for _, src := range m.srcs {
		v := src.Len()
		if v == 0 {
			// source is not yet ready
			v = 1
		}
		items += v
	}
	return items
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
