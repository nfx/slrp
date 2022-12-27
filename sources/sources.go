package sources

import (
	"time"
)

func init() {
	Sources = append(Sources, Source{
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
		ID:        19,
		Seed:      true,
		name:      "anonymous-free-proxy",
		Homepage:  "https://free-proxy-list.net/anonymous-proxy.html",
		Frequency: 30 * time.Minute,
		Feed:      httpProxyRegexFeed("https://free-proxy-list.net/anonymous-proxy.html", "Anonymous Proxy"),
	}, Source{
		ID:        21,
		name:      "uk-proxy",
		Seed:      true,
		Homepage:  "https://free-proxy-list.net/uk-proxy.html",
		Frequency: 30 * time.Minute,
		Feed:      httpProxyRegexFeed("https://free-proxy-list.net/uk-proxy.html", "UK Proxy List"),
	})
}
