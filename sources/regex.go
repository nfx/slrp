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
		regexSource(56, "https://freeproxychecker.com", ":", map[string]string{
			"http":   "/result/socks4_proxies.txt",
			"socks4": "/result/socks4_proxies.txt",
			"socks5": "/result/socks5_proxies.txt",
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
			"https":  "/https.txt",
			"socks4": "/socks4.txt",
			"socks5": "/socks5.txt",
		}, true, 1*time.Hour),
		regexSource(60, "https://api.good-proxies.ru/getfree.php", ":", map[string]string{
			"http":   "?type%5Bhttp%5D=on&anon%5B%27transparent%27%5D=on&anon%5B%27anonymous%27%5D=on&anon%5B%27elite%27%5D=on&count=100&ping=8000&time=600&works=50&key=freeproxy",
			"https":  "?type%5Bhttps%5D=on&anon%5B%27transparent%27%5D=on&anon%5B%27anonymous%27%5D=on&anon%5B%27elite%27%5D=on&count=100&ping=8000&time=600&works=50&key=freeproxy",
			"socks4": "?type%5Bsocks4%5D=on&anon%5B%27transparent%27%5D=on&anon%5B%27anonymous%27%5D=on&anon%5B%27elite%27%5D=on&count=100&ping=8000&time=600&works=50&key=freeproxy",
			"socks5": "?type%5Bsocks5%5D=on&anon%5B%27transparent%27%5D=on&anon%5B%27anonymous%27%5D=on&anon%5B%27elite%27%5D=on&count=100&ping=8000&time=600&works=50&key=freeproxy",
		}, false, 1*time.Hour),
		// https://sheesh.rip looks too shady
		// regexSource(61, "https://sheesh.rip/", ":", map[string]string{
		// 	"http":   "http",
		// 	"socks4": "socks4",
		// 	"socks5": "socks5",
		// }, true, 1*time.Hour),
		regexSource(62, "https://rootjazz.com/proxies", ":", map[string]string{
			"http": "/proxies.txt",
		}, true, 1*time.Hour),
		regexSource(63, "https://proxyscan.io/download?type=", ":", map[string]string{
			"http":   "http",
			"https":  "https",
			"socks4": "socks4",
			"socks5": "socks5",
		}, true, 1*time.Hour),
		regexSource(64, "https://www.juproxy.com", ":", map[string]string{
			"http": "/free_api",
		}, true, 1*time.Hour),
		regexSource(66, "https://www.proxyscan.io", ":", map[string]string{
			"http":   "/download?type=http",
			"https":  "/download?type=https",
			"socks4": "/download?type=socks4",
			"socks5": "/download?type=socks5",
		}, true, 1*time.Hour),
		regexSource(67, "https://openproxy.space", "Proxy List", map[string]string{
			"http":   "/list/http",
			"socks4": "/list/socks4",
			"socks5": "/list/socks5",
		}, false, 24*time.Hour),
		regexSource(68, "https://proxypedia.org", "Proxy List", map[string]string{
			"http": "/",
		}, false, 10*time.Minute),
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
