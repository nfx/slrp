package sources

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// TODO: make source name from domain name by default and refer to source by ID in state
func init() {
	Sources = append(Sources, Source{
		ID:        8,
		Homepage:  "https://free-proxy-list.net",
		Frequency: 1 * time.Hour,
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
		ID:        12,
		name:      "speedx",
		Homepage:  "https://github.com/TheSpeedX/PROXY-List",
		UrlPrefix: "https://raw.githubusercontent.com/TheSpeedX/SOCKS-List",
		Frequency: 3 * time.Hour,
		Seed:      true,
		Feed:      theSpeedX,
	}, Source{
		ID:        13,
		Homepage:  "http://proxydb.net/",
		Frequency: 6 * time.Hour,
		Feed:      proxyDb,
	}, Source{
		ID:        17,
		name:      "jetkai",
		Homepage:  "https://github.com/jetkai/proxy-list",
		UrlPrefix: "https://raw.githubusercontent.com/jetkai/proxy-list",
		Frequency: 2 * time.Hour,
		Seed:      true,
		Feed:      jetkaiProxyBuilder,
	})
}

func jetkaiProxyBuilder(ctx context.Context, h *http.Client) Src {
	// aggregated by https://github.com/jetkai/proxy-builder-2
	f := regexFeedBase(ctx, h, "https://raw.githubusercontent.com/jetkai/proxy-list", ":")
	return merged().
		refresh(f("/main/online-proxies/txt/proxies-http.txt", "http")).
		refresh(f("/main/online-proxies/txt/proxies-https.txt", "https")).
		refresh(f("/main/online-proxies/txt/proxies-socks4.txt", "socks4")).
		refresh(f("/main/online-proxies/txt/proxies-socks5.txt", "socks5"))
}

func proxylists(ctx context.Context, h *http.Client) Src {
	f := regexFeedBase(ctx, h, "http://proxylists.net", ":")
	return merged().
		refresh(f("/http_highanon.txt", "http")).
		refresh(f("/socks4.txt", "socks4")).
		refresh(f("/socks5.txt", "socks5"))
}

func theSpeedX(ctx context.Context, h *http.Client) Src {
	f := regexFeedBase(ctx, h, "https://raw.githubusercontent.com/TheSpeedX/SOCKS-List/master", ":")
	return merged().
		refresh(f("/http.txt", "http")).
		refresh(f("/socks4.txt", "socks4")).
		refresh(f("/socks5.txt", "socks5"))
}

func proxyDb(ctx context.Context, h *http.Client) Src {
	tpl := "http://proxydb.net/?country=%s&anonlvl=%s&protocol=%s"
	merged := merged()
	for _, country := range countries {
		for _, anonlvl := range []string{"4", "3", "2"} {
			for _, protocol := range []string{"http", "https", "socks4", "socks5"} {
				// it's simple enough for this horrible nesting
				url := fmt.Sprintf(tpl, country, anonlvl, protocol)
				merged.refresh(regexFeed(ctx, h, url, protocol, "Proxy List"))
			}
		}
	}
	return merged
}
