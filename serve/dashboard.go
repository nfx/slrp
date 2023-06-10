package serve

import (
	"net/http"
	"time"

	"github.com/nfx/slrp/probe"
	"github.com/nfx/slrp/refresher"
	"github.com/nfx/slrp/sorter"
	"github.com/nfx/slrp/sources"
	"github.com/nfx/slrp/stats"
)

type dashboard struct {
	refresher *refresher.Refresher
	probe     *probe.Probe
	stats     *stats.Stats
}

func NewDashboard(
	refresher *refresher.Refresher,
	probe *probe.Probe,
	stats *stats.Stats) *dashboard {
	return &dashboard{
		refresher: refresher,
		probe:     probe,
		stats:     stats,
	}
}

type Dashboard struct {
	Cards   []Card
	Refresh []src
}

type Card struct {
	Name      string
	Value     interface{}
	Increment interface{}
}

type src struct {
	Name         string
	Homepage     string
	UrlPrefix    string
	Frequency    string
	State        string
	Failure      string
	Progress     int
	Dirty        int
	Contribution int
	Exclusive    int
	Scheduled    int
	New          int
	Probing      int
	Found        int
	Timeouts     int
	Blacklisted  int
	Ignored      int
	Updated      time.Time
	EstFinish    time.Time
	NextRefresh  time.Time
}

func (d *dashboard) HttpGet(_ *http.Request) (interface{}, error) {
	probe := d.probe.Snapshot()
	exclusive := map[int]int{}
	dirty := map[int]int{}
	contribution := map[int]int{}
	for ip, v := range probe.SeenSources {
		for sid := range v {
			_, ok := probe.Blacklist[ip]
			if ok {
				continue
			}
			_, ok = probe.LastReverified[ip]
			if ok {
				continue
			}
			// dirty is dirty working proxies with dupes
			dirty[sid] += 1
		}
		if len(v) > 1 {
			continue
		}
		for sid := range v {
			// exclusive source contribution
			contribution[sid] += 1
		}
		_, ok := probe.Blacklist[ip]
		if ok {
			continue
		}
		_, ok = probe.LastReverified[ip]
		if ok {
			continue
		}
		for sid := range v {
			// exclusive source contribution to found working proxies
			exclusive[sid] += 1
		}
	}
	seenIncr := 0
	reverifyIncr := 0
	blacklistIncr := 0
	stats := d.stats.Snapshot()
	for k, v := range stats {
		if time.Since(v.Updated) < 1*time.Hour {
			seenIncr += v.Found
			blacklistIncr += v.Blacklisted
			if k != 0 {
				reverifyIncr += v.Timeouts
			}
		}
	}
	plan := d.refresher.Snapshot()
	srcs := []src{}
	for _, s := range sources.Sources {
		urlPrefix := s.UrlPrefix
		if urlPrefix == "" {
			urlPrefix = s.Homepage
		}
		stat := stats[s.ID]
		var progress int
		var delay time.Duration
		estFinish := time.Now()
		status, ok := plan[s.ID]
		if ok {
			delay = status.Delay
			progress = status.Progress()
			estFinish = status.EstFinish(stat.Scheduled + stat.New + stat.Probing)
		}
		srcs = append(srcs, src{
			Name:         s.Name(),
			Homepage:     s.Homepage,
			UrlPrefix:    urlPrefix,
			Frequency:    s.Frequency.String(),
			Dirty:        dirty[s.ID],
			Contribution: contribution[s.ID],
			Exclusive:    exclusive[s.ID],
			State:        string(stat.State),
			Failure:      stat.Failure,
			Progress:     progress,
			Scheduled:    stat.Scheduled,
			New:          stat.New,
			Probing:      stat.Probing,
			Found:        stat.Found,
			Timeouts:     stat.Timeouts,
			Blacklisted:  stat.Blacklisted,
			Ignored:      stat.Ignored,
			Updated:      stat.Updated,
			EstFinish:    estFinish,
			NextRefresh:  stat.Updated.Add(delay),
		})
	}
	reverify, ok := stats[0]
	if ok {
		srcs = append(srcs, src{
			Name:        "reverify",
			Homepage:    "/reverify",
			Frequency:   "1h",
			State:       string(reverify.State),
			Failure:     reverify.Failure,
			Progress:    reverify.Progress,
			Scheduled:   reverify.Scheduled,
			New:         reverify.New,
			Probing:     reverify.Probing,
			Found:       reverify.Found,
			Blacklisted: reverify.Blacklisted,
			Ignored:     reverify.Ignored,
			Updated:     reverify.Updated,
		})
	}
	sorter.Slice(srcs, func(i int) sorter.Cmp {
		return sorter.Chain{
			// sorter.IntAsc(srcs[i].NextRefresh.Ui),
			sorter.IntDesc(srcs[i].Contribution),
		}
	})
	return Dashboard{
		Cards: []Card{
			{"seen", len(probe.Seen), seenIncr},
			{"reverify", len(probe.LastReverified), reverifyIncr},
			{"blacklist", len(probe.Blacklist), blacklistIncr},
		},
		Refresh: srcs,
	}, nil
}
