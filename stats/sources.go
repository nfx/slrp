package stats

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type Sources map[int]Stat

func (s Sources) LastUpdated() (last time.Time) {
	for _, v := range s {
		if v.Updated.After(last) {
			last = v.Updated
		}
	}
	return
}

func (s Sources) IsRunning(source int) bool {
	stats, ok := s[source]
	if !ok {
		return false
	}
	return stats.State == Running
}

func (s Sources) String() string {
	b := []string{}
	header := fmt.Sprintf(
		"%16s\t->S\t->N\t->P\tGOOD\t  *F\t  *T\t  *B\t  *I\tUPDATED\n",
		"SOURCE")
	var found, timeouts, blacklisted, ignored int
	for k, v := range s {
		success := int(100 * float32(v.Found) / float32(v.Processed()))
		if v.Found == 0 {
			success = 0
		}
		info := fmt.Sprintf("%16d\t%3d\t%3d\t%3d\t%3d%%\t%4d\t%4d\t%4d\t%5d\t%s",
			k, v.Scheduled, v.New, v.Probing, success, v.Found, v.Timeouts,
			v.Blacklisted, v.Ignored, time.Since(v.Updated).Round(time.Second))
		b = append(b, info)
		found += v.Found
		timeouts += v.Timeouts
		blacklisted += v.Blacklisted
		ignored += v.Ignored
	}
	sort.Strings(b)
	other := fmt.Sprintf("\n%52s\t%4d\t%4d\t%4d\t%5d", "total",
		found, timeouts, blacklisted, ignored)
	return header + strings.Join(b, "\n") + other
}
