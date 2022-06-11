package pool

import (
	"testing"
	"time"
)

func TestSortStrategyPicks(t *testing.T) {
	entries := []entry{
		{
			ReanimateAfter: now(),
			LastSeen:       now().Unix(),
			Speed:          5 * time.Second,
			Offered:        1,
		},
		{
			ReanimateAfter: now(),
			LastSeen:       now().Unix(),
			Speed:          1 * time.Second,
			Offered:        50,
		},
	}
	fastestUnseen(entries)
	sortForDisplay(entries)
	saturate(entries)
}
