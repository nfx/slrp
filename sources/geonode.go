package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/nfx/slrp/pmux"
	"github.com/rs/zerolog/log"
)

var geoNodeURL = "https://proxylist.geonode.com/api/proxy-list"

type geoNodeResult struct {
	IP             string   `json:"ip"`
	Port           string   `json:"port"`
	Protocols      []string `json:"protocols"`
	AnonymityLevel string   `json:"anonymityLevel"`
	ResponseTime   int      `json:"responseTime"`
	Country        string   `json:"country"`
}

func (r *geoNodeResult) IsGood() bool {
	return r.AnonymityLevel == "elite" || r.AnonymityLevel == "anonymous"
}

func (r *geoNodeResult) Proxies() (proxies []pmux.Proxy) {
	addr := fmt.Sprintf("%s:%s", r.IP, r.Port)
	for _, v := range r.Protocols {
		proxies = append(proxies, pmux.NewProxy(addr, v))
	}
	return
}

type geoNodeResultPage struct {
	Data  []geoNodeResult `json:"data"`
	Total int             `json:"total"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
}

func init() {
	Sources = append(Sources, Source{
		ID:        23,
		Homepage:  "https://geonode.com/free-proxy-list/",
		Frequency: 3 * time.Hour,
		Seed:      true,
		Feed:      simpleGen(geoNode),
	})
}

func geoNode(ctx context.Context, h *http.Client) (found []pmux.Proxy, err error) {
	qs := url.Values{}
	qs.Set("sort_by", "lastChecked")
	qs.Set("sort_type", "desc")
	qs.Set("limit", "500")
	page := 1
	var results geoNodeResultPage
	for {
		log.Info().Int("page", page).Msg("Loading geocode database")
		qs.Set("page", fmt.Sprint(page))
		r, err := h.Get(geoNodeURL + "?" + qs.Encode())
		if err != nil {
			return nil, err
		}
		raw, err := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(raw, &results)
		if err != nil {
			return nil, err
		}
		if len(results.Data) == 0 {
			break
		}
		page = results.Page + 1
		for _, d := range results.Data {
			if !d.IsGood() {
				continue
			}
			found = append(found, d.Proxies()...)
		}
	}
	return
}
