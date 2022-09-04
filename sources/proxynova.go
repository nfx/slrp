package sources

import (
	"context"
	"encoding/base64"
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

func documentWriteInJsVm(ctx context.Context, script string) (string, error) {
	vm := goja.New()
	vm.GlobalObject().Set("atob", func(in string) (string, error) {
		dec, err := base64.StdEncoding.DecodeString(in)
		return string(dec), err
	})
	vm.RunString(`var document = {
		write: function(str) {
			return str;
		}
	};`)
	var wrote string
	v, err := vm.RunString(script)
	if err != nil {
		return "", err
	}
	err = vm.ExportTo(v, &wrote)
	return wrote, err
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
			err = p.Each2("Proxy IP", "Proxy Port", func(ip, port string) error {
				if !strings.Contains(ip, "document") {
					return nil
				}
				proxy, err := documentWriteInJsVm(ctx, ip)
				if proxy == "" {
					log.Err(err).Str("src", ip).Msg("failed to execute javascript")
					return err
				}
				port = strings.ReplaceAll(port, ".0", "")
				found = append(found, pmux.HttpProxy(proxy+":"+port))
				return nil
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
