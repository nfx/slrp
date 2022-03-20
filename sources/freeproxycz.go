package sources

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pmux"
)

func init() {
	Sources = append(Sources, Source{
		ID:        2,
		Homepage:  "http://free-proxy.cz/",
		Frequency: 1 * time.Hour,
		Feed:      freeProxyCz,
	})
}

// Scrapes http://free-proxy.cz/
func freeProxyCz(ctx context.Context, h *http.Client) Src {
	b64 := base64.StdEncoding
	//look for "Web Proxy List"
	log := app.Log.From(ctx)
	fetch := func(url string) func() ([]pmux.Proxy, error) {
		return func() (found []pmux.Proxy, err error) {
			p, serial, err := newTablePage(ctx, h, url, "Web Proxy List")
			if err != nil {
				return
			}
			err = p.Each3("IP address", "Port", "Protocol", func(a, b, c string) {
				enc := strings.Split(a, `"`)
				if len(enc) != 3 {
					return
				}
				addr, err := b64.DecodeString(enc[1])
				if err != nil {
					log.Warn().Err(err).Msg("cannot demangle ip")
				}
				proxy := pmux.NewProxy(fmt.Sprintf("%s:%s", addr, b), strings.ToLower(c))
				found = append(found, proxy)
			})
			if err != nil {
				err = skipErr(err, intEC{"serial", serial}, strEC{"url", url})
			}
			return
		}
	}
	pattern := "http://free-proxy.cz/en/proxylist/main/date/%d"
	merged := merged()
	for i := 1; i < 23; i++ {
		url := fmt.Sprintf(pattern, i)
		merged.refresh(fetch(url))
	}
	tpl := "http://free-proxy.cz/en/proxylist/country/%s/%s/ping/%s"
	for _, country := range countries {
		for _, anonlvl := range []string{"level1", "level2"} {
			for _, protocol := range []string{"http", "https", "socks4", "socks5"} {
				// it's simple enough for this horrible nesting
				url := fmt.Sprintf(tpl, country, protocol, anonlvl)
				merged.refresh(fetch(url))
			}
		}
	}
	return merged
}
