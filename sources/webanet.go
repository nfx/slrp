package sources

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/nfx/slrp/pmux"
)

func init() {
	Sources = append(Sources, Source{
		ID:        16,
		Homepage:  "https://webanetlabs.net/publ/24",
		UrlPrefix: "https://webanetlabs.net",
		Frequency: 24 * time.Hour,
		Feed:      webanet,
	})
}

type webaNet struct {
	h   *http.Client
	out chan pmux.Proxy
	err error
	src Src
}

func (w *webaNet) Generate(ctx context.Context) <-chan pmux.Proxy {
	merged := merged()
	w.out = make(chan pmux.Proxy)
	w.src = merged
	recent, err := findLinksWithOn(ctx, w.h,
		fmt.Sprintf("/proxylist%d", time.Now().Year()),
		"https://webanetlabs.net/publ/24")
	if err != nil {
		defer close(w.out)
		w.err = err
		return w.out
	}
	if len(recent) == 0 {
		defer close(w.out)
		w.err = fmt.Errorf("no links found")
		return w.out
	}
	for _, v := range recent {
		merged.refresh(func() ([]pmux.Proxy, error) {
			return newRegexPage(ctx, w.h, v, "Список прокси",
				func(proxy string) pmux.Proxy {
					return pmux.HttpProxy(proxy)
				})
		})
	}
	go w.foward(ctx)
	return w.out
}

func (w *webaNet) foward(ctx context.Context) {
	defer close(w.out)
	for proxy := range w.src.Generate(ctx) {
		select {
		case w.out <- proxy:
		case <-ctx.Done():
			return
		}
	}
}

func (w *webaNet) Err() error {
	return w.err
}

func webanet(ctx context.Context, h *http.Client) Src {
	return &webaNet{
		h: h,
	}
}
