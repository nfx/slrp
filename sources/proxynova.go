package sources

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/nfx/slrp/app"
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
	log := app.Log.From(ctx)
	page := func(path string) func() ([]pmux.Proxy, error) {
		return func() (found []pmux.Proxy, err error) {
			url := proxyNovaPrefix + path
			p, serial, err := newTablePage(ctx, h, url, "Proxy Server List")
			if err != nil {
				return
			}
			vm := goja.New() // TODO: implement atob() in go
			_, err = vm.RunString(`
			var document = {
				write: function(str) {
					return str;
				}
			};`)
			if err != nil {
				return nil, err
			}
			err = p.Each2("Proxy IP", "Proxy Port", func(ip, port string) {
				if !strings.Contains(ip, "document") {
					return
				}
				var proxy string
				v, err := vm.RunString(ip)
				if err != nil {
					log.Err(err).Str("src", ip).Msg("failed to execute javascript")
					return
				}
				err = vm.ExportTo(v, &proxy)
				if err != nil {
					return
				}
				if proxy == "" {
					return
				}
				port = strings.ReplaceAll(port, ".0", "")
				ip = strings.ReplaceAll(proxy, "' + '", "")
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
