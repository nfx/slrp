package sources

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pmux"
)

func init() {
	Sources = append(Sources, Source{
		ID:        6,
		Homepage:  "https://proxy-list.org/",
		Frequency: 5 * time.Minute,
		Feed:      proxyListOrg,
	})
}

var proxyListOrgRE = regexp.MustCompile(`(?m)Proxy\('([^']+)'\)`)

func proxyListOrg(ctx context.Context, h *http.Client) Src {
	merged := merged()
	b64 := base64.StdEncoding
	log := app.Log.From(ctx)
	x := func(url string) func() ([]pmux.Proxy, error) {
		return func() ([]pmux.Proxy, error) {
			body, _, err := req{URL: url}.Do(ctx, h)
			if err != nil {
				return nil, err
			}
			found := []pmux.Proxy{}
			for _, match := range proxyListOrgRE.FindAllSubmatch(body, -1) {
				a := strings.Trim(string(match[1]), `"`)
				addr, err := b64.DecodeString(a)
				if err != nil {
					log.Warn().Err(err).Msg("cannot demangle ip")
				}
				found = append(found, pmux.NewProxy(string(addr), "http"))
				found = append(found, pmux.NewProxy(string(addr), "https"))
			}
			return found, nil
		}
	}
	list := "https://proxy-list.org/english/index.php?p=%d"
	for i := 1; i <= 10; i++ {
		url := fmt.Sprintf(list, i)
		merged.refresh(x(url))
	}
	return merged
}
