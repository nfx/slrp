package sources

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

var proxyDbPages = map[string]string{}

func init() {
	Sources = append(Sources, Source{
		ID:        13,
		Homepage:  "http://proxydb.net/",
		Frequency: 6 * time.Hour,
		Feed:      proxyDb,
	})
	tpl := "http://proxydb.net/?country=%s&anonlvl=%s&protocol=%s"
	for _, country := range countries {
		for _, anonlvl := range []string{"4", "3", "2"} {
			for _, protocol := range []string{"http", "https", "socks4", "socks5"} {
				// it's simple enough for this horrible nesting
				url := fmt.Sprintf(tpl, country, anonlvl, protocol)
				proxyDbPages[url] = protocol
			}
		}
	}
}

func proxyDb(ctx context.Context, h *http.Client) Src {
	merged := merged()
	for url, protocol := range proxyDbPages {
		merged.refresh(regexFeed(ctx, h, url, protocol, "Proxy List"))
	}
	return merged
}
