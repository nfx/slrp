package sources

import (
	"context"
	"net/http"
	"time"
)

func init() {
	Sources = append(Sources, Source{
		ID:        8,
		Homepage:  "https://free-proxy-list.net",
		Frequency: 30 * time.Minute,
		Feed:      httpProxyRegexFeed("https://free-proxy-list.net", "Proxy List"),
	}, Source{
		ID:        9,
		Homepage:  "http://foxtools.ru/",
		UrlPrefix: "http://api.foxtools.ru",
		Frequency: 12 * time.Hour,
		Feed:      httpProxyRegexFeed("http://api.foxtools.ru/v2/Proxy.txt", "1 1"),
	}, Source{
		ID:        10,
		name:      "sunny9577",
		Homepage:  "https://github.com/sunny9577/proxy-scraper",
		UrlPrefix: "https://sunny9577.github.io/",
		Frequency: 3 * time.Hour,
		Seed:      true,
		Feed:      httpProxyRegexFeed("https://sunny9577.github.io/proxy-scraper/proxies.txt", ":"),
	}, Source{
		ID:        11,
		Homepage:  "http://proxylists.net",
		Frequency: 3 * time.Hour,
		Seed:      true,
		Feed:      proxylists,
	}, Source{
		ID:        18,
		Homepage:  "https://sslproxies.org/",
		Frequency: 30 * time.Minute,
		Feed:      sslProxies,
	}, Source{
		ID:        19,
		name:      "anonymous-free-proxy",
		Homepage:  "https://free-proxy-list.net/anonymous-proxy.html",
		Frequency: 30 * time.Minute,
		Feed:      httpProxyRegexFeed("https://free-proxy-list.net/anonymous-proxy.html", "Anonymous Proxy"),
	}, Source{
		ID:        20,
		Homepage:  "https://us-proxy.org",
		Frequency: 30 * time.Minute,
		Feed:      httpProxyRegexFeed("https://www.us-proxy.org", "US Proxy List"),
	}, Source{
		ID:        21,
		name:      "uk-proxy",
		Homepage:  "https://free-proxy-list.net/uk-proxy.html",
		Frequency: 30 * time.Minute,
		Feed:      httpProxyRegexFeed("https://free-proxy-list.net/uk-proxy.html", "UK Proxy List"),
	})
}

func sslProxies(ctx context.Context, h *http.Client) Src {
	f := regexFeedBase(ctx, h, "http://sslproxies.org/", "SSL Proxy List")
	return merged().refresh(f("/", "https"))
}

func proxylists(ctx context.Context, h *http.Client) Src {
	f := regexFeedBase(ctx, h, "http://proxylists.net", ":")
	return merged().
		refresh(f("/http_highanon.txt", "http")).
		refresh(f("/socks4.txt", "socks4")).
		refresh(f("/socks5.txt", "socks5"))
}
