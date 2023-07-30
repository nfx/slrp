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
	out chan Signal
	err error
	src Src
}

var webanetURL = "https://webanetlabs.net/publ/24"

func (w *webaNet) Generate(ctx context.Context) <-chan Signal {
	merged := merged()
	w.out = make(chan Signal)
	w.src = merged
	recent, err := findLinksWithOn(ctx, w.h,
		fmt.Sprintf("/proxylist%d", time.Now().Year()),
		webanetURL)
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
		page := v // that nasty go iteration gotcha...
		merged.refresh(func() ([]pmux.Proxy, error) {
			return newRegexPage(ctx, w.h, page, "Список прокси",
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

func (w *webaNet) Len() int {
	return w.src.Len()
}

func webanet(ctx context.Context, h *http.Client) Src {
	return &webaNet{
		h: h,
	}
}
