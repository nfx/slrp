package probe

import (
	"fmt"
	"net/http"
	"sort"

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
	Proxy   string
	Failure string
	Sources []string

	failureIndex int
}

type Card struct {
	Name  string
	Value int
}

type blacklistedResults struct {
	TopFailures []Card
	TopSources  []Card
	Items       []blacklisted
}

func (d *dashboard) HttpGet(r *http.Request) (interface{}, error) {
	probe := d.probe.Snapshot()
	snapshot := []blacklisted{}
	for proxyUint32, failureIndex := range probe.Blacklist {
		srcs := []string{}
		for src := range probe.SeenSources[proxyUint32] {
			srcs = append(srcs, sources.ByID(src).Name())
		}
		snapshot = append(snapshot, blacklisted{
			Proxy:   fmt.Sprintf("%d", proxyUint32), // TODO: fix it
			Failure: probe.Failures[failureIndex],
			Sources: srcs,

			failureIndex: failureIndex,
		})
	}
	if len(snapshot) == 0 {
		return nil, fmt.Errorf("blacklist is empty")
	}
	filter := r.FormValue("filter")
	query, err := ql.Parse[blacklisted](filter)
	if err != nil {
		return nil, err
	}
	// if len(query.OrderBy) == 0 {
	// 	query.OrderBy = []ql.OrderBy{
	// 		ql.Asc("Failure"),
	// 	}
	// }
	result := blacklistedResults{}
	failureSummary := map[int]int{}
	srcSummary := map[string]int{}
	err = query.ApplyFacets(&snapshot, &result.Items, func(all *[]blacklisted) {
		for _, v := range *all {
			failureSummary[v.failureIndex]++
			for _, src := range v.Sources {
				srcSummary[src]++
			}
		}
	})
	for failureIndex, cnt := range failureSummary {
		result.TopFailures = append(result.TopFailures, Card{
			Name:  probe.Failures[failureIndex],
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
