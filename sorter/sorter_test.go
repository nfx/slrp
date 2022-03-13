package sorter

import (
	"log"
	"testing"
)

type x struct {
	first, second, third int
}

// TODO: benchmark test between Chain & Chain2

func Test(t *testing.T) {
	fixture := []x{
		{1, 6, 10},
		{1, 5, 9},
		{1, 5, 8},
		{2, 4, 7},
		{2, 3, 6},
		{2, 3, 5},
		{3, 2, 4},
		{3, 1, 3},
		{3, 1, 2},
	}
	Slice(fixture, func(i int) Cmp {
		return Chain2{
			IntDesc(fixture[i].first),
			IntDesc(fixture[i].second),
			IntAsc(fixture[i].third),
		}
	})
	log.Printf("%v", fixture)
}
