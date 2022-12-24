package sources

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nfx/slrp/pmux"
)

var hidemyNamePages []string

func init() {
	Sources = append(Sources, Source{
		ID:        24,
		Homepage:  "https://hidemy.name",
		Frequency: 1 * time.Hour,
		Feed:      hidemyName,
	})
	// https://hidemy.name/en/proxy-list/?anon=34
	hidemyNamePages = []string{}
	// 64 per page
	pattern := "https://hidemy.name/en/proxy-list/?anon=34&start=%d"
	// this source has approx 500 high anon proxies
	for i := 0; i < 500; i += 64 {
		url := fmt.Sprintf(pattern, i)
		hidemyNamePages = append(hidemyNamePages, url)
	}
}

// Scrapes http://hidemy.name/
func hidemyName(ctx context.Context, h *http.Client) Src {
	fetch := func(url string) func() ([]pmux.Proxy, error) {
		return func() (found []pmux.Proxy, err error) {
			p, serial, err := newTablePage(ctx, h, url, "Online database of proxy lists")
			if err != nil {
				return
			}
			err = p.Each3("IP address", "Port", "Type", func(host, port, types string) error {
				for _, v := range strings.Split(types, ",") {
					v = strings.ToLower(strings.TrimSpace(v))
					proxy := pmux.NewProxy(fmt.Sprintf("%s:%s", host, port), v)
					found = append(found, proxy)
				}
				return nil
			})
			if err != nil {
				err = skipErr(err, intEC{"serial", serial}, strEC{"url", url})
			}
			return
		}
	}
	merged := merged()
	for _, url := range hidemyNamePages {
		merged.refresh(fetch(url))
	}
	return merged
}
