package stats

import (
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
