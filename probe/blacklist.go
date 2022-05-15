package probe

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/nfx/slrp/ipinfo"
	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/ql"
	"github.com/nfx/slrp/sources"
)

type dashboard struct {
	probe *Probe
	info  *ipinfo.Lookup
}

func NewBlacklistApi(probe *Probe, info *ipinfo.Lookup) *dashboard {
	return &dashboard{
		probe: probe,
		info:  info,
	}
}

type blacklisted struct {
	Proxy    pmux.Proxy
	Country  string
	Provider string
	ASN      uint16
	Failure  string
	Sources  []string
}

type Card struct {
	Name  string
	Value int
}

type blacklistedResults struct {
	Total        int
	TopFailures  []Card
	TopSources   []Card
	TopCountries []Card
	TopProviders []Card
	Items        []blacklisted
}

func (d *dashboard) HttpGet(r *http.Request) (interface{}, error) {
	probe := d.probe.Snapshot()
	snapshot := []blacklisted{}
	for proxy, failureIndex := range probe.Blacklist {
		srcs := []string{}
		for src := range probe.SeenSources[proxy] {
			srcs = append(srcs, sources.ByID(src).Name())
		}
		info := d.info.Get(proxy)
		snapshot = append(snapshot, blacklisted{
			Failure:  probe.Failures[failureIndex],
			Country:  info.Country,
			Provider: info.Provider,
			ASN:      info.ASN,
			Proxy:    proxy,
			Sources:  srcs,
		})
	}
	if len(snapshot) == 0 {
		return nil, fmt.Errorf("blacklist is empty")
	}
	filter := r.FormValue("filter")
	result := blacklistedResults{}
	srcSummary := Summary{}
	failureSummary := Summary{}
	countrySummary := Summary{}
	providerSummary := Summary{}
	err := ql.Execute(&snapshot, &result.Items, filter, func(all *[]blacklisted) {
		result.Total = len(*all)
		for _, v := range *all {
			split := strings.Split(v.Failure, ": ")
			// perform error common suffix normalisation
			failure := split[len(split)-1]

			// TODO: facetisation seems soo common, that it could be reflected through fieldMap
			failureSummary[failure]++
			countrySummary[v.Country]++
			providerSummary[v.Provider]++
			for _, src := range v.Sources {
				srcSummary[src]++
			}
		}
	}, ql.DefaultLimit(50), ql.DefaultOrder{ql.Asc("Proxy")})
	if err != nil {
		return nil, err
	}
	result.TopFailures = failureSummary.TopN(10)
	result.TopSources = srcSummary.TopN(10)
	result.TopCountries = countrySummary.TopN(10)
	result.TopProviders = providerSummary.TopN(10)
	return result, err
}

type Summary map[string]int

func (s Summary) TopN(n int) (cards []Card) {
	for name, cnt := range s {
		if name == "" {
			name = "n/a"
		}
		cards = append(cards, Card{
			Name:  name,
			Value: cnt,
		})
	}
	topNCards(n, &cards)
	return cards
}

func topNCards(min int, cards *[]Card) {
	sort.Slice(*cards, func(i, j int) bool {
		return (*cards)[i].Value > (*cards)[j].Value
	})
	failuresLen := len(*cards)
	if failuresLen < min {
		min = failuresLen
	}
	*cards = (*cards)[0:min]
}
