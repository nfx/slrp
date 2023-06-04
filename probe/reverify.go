package probe

import (
	"fmt"
	"net/http"
	"time"

	"github.com/nfx/slrp/ipinfo"
	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/sources"
)

type probeSnapshot interface {
	Snapshot() internal
}

type reverifyDashboard struct {
	Probe  probeSnapshot
	Lookup ipinfo.IpInfoGetter
}

func NewReverifyApi(probe *Probe, info *ipinfo.Lookup) *reverifyDashboard {
	return &reverifyDashboard{
		Probe:  probe,
		Lookup: info,
	}
}

//go:generate go run ../ql/generator/main.go inReverify
type inReverify struct {
	Proxy    pmux.Proxy
	Attempt  int
	After    time.Time
	Country  string `facet:"Country"`
	Provider string `facet:"Provider"`
	ASN      uint16
	Failure  string `facet:"Failure"`
	Sources  []string
}

func (d *reverifyDashboard) snapshot() (found inReverifyDataset) {
	s := d.Probe.Snapshot()
	for _, v := range s.LastReverified {
		info := d.Lookup.Get(v.Proxy)
		srcs := []string{}
		for src := range s.SeenSources[v.Proxy] {
			srcs = append(srcs, sources.ByID(src).Name())
		}
		found = append(found, inReverify{
			Proxy:    v.Proxy,
			Attempt:  v.Attempt,
			After:    time.Unix(v.After, 0),
			Country:  info.Country,
			Provider: info.Provider,
			ASN:      info.ASN,
			Sources:  srcs,
		})
	}
	return found
}

func (d *reverifyDashboard) HttpGet(r *http.Request) (any, error) {
	snapshot := d.snapshot()
	if len(snapshot) == 0 {
		return nil, fmt.Errorf("reverify is empty")
	}
	return snapshot.Query(r.FormValue("filter"))
}
