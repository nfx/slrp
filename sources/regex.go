package sources

import (
	"context"
	"net/http"
	"time"
)

func init() {
	Sources = append(Sources,
		regexSource(8, "https://free-proxy-list.net", "Proxy List", map[string]string{
			"http": "/",
		}, false, 30*time.Minute),
		regexSource(11, "http://proxylists.net", ":", map[string]string{
			"http":   "/http_highanon.txt",
			"socks4": "/socks4.txt",
			"socks5": "/socks5.txt",
		}, true, 3*time.Hour),
		regexSource(18, "https://sslproxies.org", "SSL Proxy List", map[string]string{
			"https": "/",
		}, false, 30*time.Minute),
		regexSource(20, "https://us-proxy.org", "US Proxy List", map[string]string{
			"http": "/",
		}, false, 30*time.Minute),
		regexSource(55, "https://proxy-list.download", ":", map[string]string{
			"http":   "/api/v1/get?type=http",
			"https":  "/api/v1/get?type=https",
			"socks4": "/api/v1/get?type=socks4",
			"socks5": "/api/v1/get?type=socks5",
		}, true, 3*time.Hour),
	)
}

func regexSource(id int, home, expect string, files map[string]string, seed bool, freq time.Duration) Source {
	return Source{
		ID:        id,
		Homepage:  home,
		Frequency: freq,
		Seed:      seed,
		Feed: func(ctx context.Context, h *http.Client) Src {
			f := regexFeedBase(ctx, h, home, expect)
			m := merged()
			for t, loc := range files {
				m.refresh(f(loc, t))
			}
			return m
		},
	}
}
