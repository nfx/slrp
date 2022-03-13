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
	log := app.Log.From(ctx)
	pattern := "http://free-proxy.cz/en/proxylist/main/date/%d"
	merged := merged()
	for i := 1; i < 23; i++ {
		url := fmt.Sprintf(pattern, i)
		merged.refresh(func() (found []pmux.Proxy, err error) {
			p, err := newTablePage(ctx, h, url)
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
				err = fmt.Errorf("issue with %s: %s", url, err)
			}
			return
		})
	}
	return merged
}
