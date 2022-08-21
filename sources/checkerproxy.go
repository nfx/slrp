package sources

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/nfx/slrp/pmux"

	"github.com/rs/zerolog/log"
)

var checkerProxyURL = "https://checkerproxy.net/"

func init() {
	Sources = append(Sources, Source{
		ID:        1,
		Homepage:  "https://checkerproxy.net/",
		Frequency: 6 * time.Hour,
		Seed:      true,
		Feed:      simpleGen(checkerProxy),
	})
}

// Scrapes https://checkerproxy.net/
func checkerProxy(ctx context.Context, h *http.Client) (found []pmux.Proxy, err error) {
	log.Info().Msg("Loading proxy checker database")
	archives, err := findLinksWithOn(ctx, h, "/archive", checkerProxyURL)
	if err != nil {
		return
	}
	proxyTypes := map[int]string{
		1: "http",
		2: "https",
		4: "socks5",
	}
	type info struct {
		Addr    string `json:"addr"`
		Kind    int    `json:"kind"`
		Timeout int    `json:"timeout"`
		Type    int    `json:"type"`
	}
	for _, v := range archives[0:1] { // only first archive
		url := strings.ReplaceAll(v, "/archive", "/api/archive")
		body, _, err := req{URL: url}.Do(ctx, h)
		if err != nil {
			return nil, err
		}
		log.Info().Str("url", url).Msg("done loading")
		var items []info
		err = json.Unmarshal(body, &items)
		if err != nil {
			return nil, err
		}
		for _, proxy := range items {
			proxyType, ok := proxyTypes[proxy.Type]
			if !ok {
				log.Warn().Int("proxyType", proxy.Type).Msg("new proxy type")
				continue
			}
			if proxy.Kind != 2 {
				// not anonymous
				continue
			}
			if proxy.Timeout > 10000 {
				// more than 10s timeout
				continue
			}
			found = append(found, pmux.NewProxy(proxy.Addr, proxyType))
		}
	}
	return found, nil
}
