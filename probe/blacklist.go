package probe

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/ql"
	"github.com/nfx/slrp/sources"
)

type dashboard struct {
	probe *Probe
}

func NewBlacklistApi(probe *Probe) *dashboard {
	return &dashboard{
		probe: probe,
	}
}

type blacklisted struct {
	Proxy   pmux.Proxy
	Failure string
	Sources []string
}

type Card struct {
	Name  string
	Value int
}

type blacklistedResults struct {
	Total       int
	TopFailures []Card
	TopSources  []Card
	Items       []blacklisted
}

func (d *dashboard) HttpGet(r *http.Request) (interface{}, error) {
	probe := d.probe.Snapshot()
	snapshot := []blacklisted{}
	for proxy, failureIndex := range probe.Blacklist {
		srcs := []string{}
		for src := range probe.SeenSources[proxy] {
			srcs = append(srcs, sources.ByID(src).Name())
		}
		snapshot = append(snapshot, blacklisted{
			Failure: probe.Failures[failureIndex],
			Proxy:   proxy,
			Sources: srcs,
		})
	}
	if len(snapshot) == 0 {
		return nil, fmt.Errorf("blacklist is empty")
	}
	filter := r.FormValue("filter")
	result := blacklistedResults{}
	srcSummary := map[string]int{}
	failureSummary := map[string]int{}
	err := ql.Execute(&snapshot, &result.Items, filter, func(all *[]blacklisted) {
		result.Total = len(*all)
		for _, v := range *all {
			split := strings.Split(v.Failure, ": ")
			// perform error common suffix normalisation
			failure := split[len(split)-1]
			failureSummary[failure]++
			for _, src := range v.Sources {
				srcSummary[src]++
			}
		}
	}, ql.DefaultLimit(50), ql.DefaultOrder{ql.Asc("Proxy")})
	if err != nil {
		return nil, err
	}
	for failure, cnt := range failureSummary {
		result.TopFailures = append(result.TopFailures, Card{
			Name:  failure,
			Value: cnt,
		})
	}
	topNCards(10, &result.TopFailures)
	for src, cnt := range srcSummary {
		result.TopSources = append(result.TopSources, Card{
			Name:  src,
			Value: cnt,
		})
	}
	topNCards(10, &result.TopSources)
	return result, err
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
