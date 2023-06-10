package probe

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/nfx/slrp/ipinfo"
	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/sources"
)

type blacklistDashboard struct {
	probe *Probe
	info  *ipinfo.Lookup
}

func NewBlacklistApi(probe *Probe, info *ipinfo.Lookup) *blacklistDashboard {
	return &blacklistDashboard{
		probe: probe,
		info:  info,
	}
}

//go:generate go run ../ql/generator/main.go blacklisted
type blacklisted struct {
	Proxy    pmux.Proxy
	Country  string `facet:"Country"`
	Provider string `facet:"Provider"`
	ASN      uint16
	Failure  string `facet:"Failure"`
	Sources  []string
}

func (d blacklistedDataset) getFailureFacet(record int) string {
	split := strings.Split(d[record].Failure, ": ")
	// perform error common suffix normalisation
	failure := split[len(split)-1]
	return failure
}

func (d *blacklistDashboard) HttpGet(r *http.Request) (interface{}, error) {
	probe := d.probe.Snapshot()
	snapshot := blacklistedDataset{}
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
	return snapshot.Query(r.FormValue("filter"))
}
