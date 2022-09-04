package sources

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nfx/slrp/pmux"
)

var freeProxyCzPages []string

func init() {
	Sources = append(Sources, Source{
		ID:        2,
		Homepage:  "http://free-proxy.cz/",
		Frequency: 1 * time.Hour,
		Feed:      freeProxyCz,
	})

	freeProxyCzPages = []string{}
	pattern := "http://free-proxy.cz/en/proxylist/main/date/%d"
	for i := 1; i < 23; i++ {
		url := fmt.Sprintf(pattern, i)
		freeProxyCzPages = append(freeProxyCzPages, url)
	}
	tpl := "http://free-proxy.cz/en/proxylist/country/%s/%s/ping/%s"
	for _, country := range countries {
		for _, anonlvl := range []string{"level1", "level2"} {
			for _, protocol := range []string{"http", "https", "socks4", "socks5"} {
				// it's simple enough for this horrible nesting
				url := fmt.Sprintf(tpl, country, protocol, anonlvl)
				freeProxyCzPages = append(freeProxyCzPages, url)
			}
		}
	}
}

// Scrapes http://free-proxy.cz/
func freeProxyCz(ctx context.Context, h *http.Client) Src {
	b64 := base64.StdEncoding
	//look for "Web Proxy List"
	fetch := func(url string) func() ([]pmux.Proxy, error) {
		return func() (found []pmux.Proxy, err error) {
			p, serial, err := newTablePage(ctx, h, url, "Web Proxy List")
			if err != nil {
				return
			}
			err = p.Each3("IP address", "Port", "Protocol", func(a, b, c string) error {
				enc := strings.Split(a, `"`)
				if len(enc) != 3 {
					return fmt.Errorf("mangled address: %s", a)
				}
				addr, err := b64.DecodeString(enc[1])
				if err != nil {
					return err
				}
				proxy := pmux.NewProxy(fmt.Sprintf("%s:%s", addr, b), strings.ToLower(c))
				found = append(found, proxy)
				return nil
			})
			if err != nil {
				err = skipErr(err, intEC{"serial", serial}, strEC{"url", url})
			}
			return
		}
	}
	merged := merged()
	for _, url := range freeProxyCzPages {
		merged.refresh(fetch(url))
	}
	return merged
}
