package sources

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

func init() {
	Sources = append(Sources, Source{
		ID:        3,
		Homepage:  "https://my-proxy.com/",
		Frequency: 1 * time.Hour,
		Feed:      myProxyCom,
	})
}

// Scrapes https://www.my-proxy.com/
func myProxyCom(ctx context.Context, h *http.Client) Src {
	f := regexFeedBase(ctx, h, "https://my-proxy.com")
	merged := merged().
		refresh(f("/free-elite-proxy.html", "http")).
		refresh(f("/free-anonymous-proxy.html", "http")).
		refresh(f("/free-socks-4-proxy.html", "socks4")).
		refresh(f("/free-socks-5-proxy.html", "socks5"))
	for i := 1; i <= 10; i++ {
		merged.refresh(f(fmt.Sprintf("/free-proxy-list-%d.html", i), "http"))
	}
	return merged
}
