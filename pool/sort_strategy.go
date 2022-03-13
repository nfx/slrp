package pool

import (
	"time"

	"github.com/nfx/slrp/sorter"
)

var defaultSorting = fastestUnseen

func fastestUnseen(entries []entry) {
	sorter.Slice(entries, func(i int) sorter.Cmp {
		e := &entries[i]
		return sorter.Chain{
			// todo: FloatDesc on HourProbability?... no data == 100% perhaps?
			sorter.IntAsc(e.ReanimateAfter.Unix()),
			sorter.IntAsc(e.Offered),
			sorter.IntDesc(e.SinceLastSeen().Round(time.Minute)),
			sorter.IntAsc(e.Speed.Round(time.Second)),
		}
	})
}

func sortForDisplay(entries []entry) {
	sorter.Slice(entries, func(i int) sorter.Cmp {
		e := &entries[i]
		return sorter.Chain{
			// todo: FloatDesc on HourProbability?... no data == 100% perhaps?
			sorter.IntAsc(e.ReanimateAfter.Unix()),
			sorter.FloatDesc(e.SuccessRate()),
			sorter.IntDesc(e.Offered),
			sorter.IntAsc(e.Speed.Round(time.Second)),
		}
	})
}

func saturate(entries []entry) {
	sorter.Slice(entries, func(i int) sorter.Cmp {
		e := &entries[i]
		return sorter.Chain{
			sorter.IntAsc(e.ReanimateAfter.Unix()),
			sorter.FloatDesc(e.SuccessRate()),
			sorter.IntDesc(e.SinceLastSeen().Round(time.Minute)),
			// sorter.IntAsc(e.Offered),
			sorter.IntAsc(e.Speed.Round(time.Second)),
		}
	})
}
