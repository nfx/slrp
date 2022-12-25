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
		regexSource(56, "https://freeproxychecker.com/result", ":", map[string]string{
			"http":   "/socks4_proxies.txt",
			"socks4": "/socks4_proxies.txt",
			"socks5": "/socks5_proxies.txt",
		}, true, 12*time.Hour),
		regexSource(57, "https://openproxylist.xyz", ":", map[string]string{
			"http":   "/http.txt",
			"socks4": "/socks4.txt",
			"socks5": "/socks5.txt",
		}, true, 1*time.Hour),
		regexSource(58, "https://api.proxyscrape.com/v2/?request=getproxies&protocol=", ":", map[string]string{
			"http":   "http",
			"socks4": "socks4",
			"socks5": "socks5",
		}, true, 1*time.Hour),
		regexSource(59, "https://proxyspace.pro", ":", map[string]string{
			"http":   "/http.txt",
			"https": "/https.txt",
			"socks4": "/socks4.txt",
			"socks5": "/socks5.txt",
		}, true, 1*time.Hour),
		regexSource(60, "https://api.good-proxies.ru/getfree.php", ":", map[string]string{
			// ToDo add support for the rest of types, 2 requests per 5 minuts on api
			"http":   "?count=1000&ping=8000&time=600&works=500&key=freeprox",
		}, true, 1*time.Hour),
		regexSource(61, "https://sheesh.rip/", ":", map[string]string{
			"http":   "http",
			"socks4": "socks4",
			"socks5": "socks5",
		}, true, 1*time.Hour),
		regexSource(62, "https://rootjazz.com/proxies", ":", map[string]string{
			"http":   "/proxies.txt",
		}, true, 1*time.Hour),
		regexSource(63, "https://proxyscan.io/download?type=", ":", map[string]string{
			"http":   "http",
			"https":   "https",
			"socks4": "socks4",
			"socks5": "socks5",
		}, true, 1*time.Hour),
		regexSource(64, "https://www.juproxy.com", ":", map[string]string{
			"http":   "/free_api",
		}, true, 1*time.Hour),
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
