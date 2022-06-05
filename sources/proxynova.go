package sources

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nfx/slrp/pmux"
)

var proxyNovaPrefix = "https://proxynova.com/proxy-server-list"

func init() {
	Sources = append(Sources, Source{
		ID:        7,
		Homepage:  "http://proxynova.com/",
		UrlPrefix: "https://proxynova.com",
		Frequency: 6 * time.Hour,
		Feed:      proxyNova,
	})
}

// Scrapes https://www.proxynova.com
func proxyNova(ctx context.Context, h *http.Client) Src {
	page := func(path string) func() ([]pmux.Proxy, error) {
		return func() (found []pmux.Proxy, err error) {
			url := proxyNovaPrefix + path
			p, serial, err := newTablePage(ctx, h, url, "Proxy Server List")
			if err != nil {
				return
			}
			err = p.Each2("Proxy IP", "Proxy Port", func(ip, port string) {
				if !strings.Contains(ip, "document") {
					return
				}
				ip = ip[16 : len(ip)-3]
				ip = strings.ReplaceAll(ip, "' + '", "")
				port = strings.ReplaceAll(port, ".0", "")
				found = append(found, pmux.HttpProxy(ip+":"+port))
			})
			if err != nil {
				err = skipErr(err, intEC{"serial", serial}, strEC{"url", url})
			}
			return
		}
	}
	merged := merged()
	for _, country := range countries {
		merged.refresh(page(fmt.Sprintf("/country-%s", strings.ToLower(country))))
	}
	return merged.
		refresh(page("/elite-proxies")).
		refresh(page("/anonymous-proxies"))
}
