package refresher

import (
	"testing"

	"github.com/nfx/slrp/sources"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	sources.Sources = []sources.Source{}

	ref, run, stop := start()
	defer stop()
	run["stats"].Spin()
	run["pool"].Spin()

	<-run["refresher"].Wait
	log.Trace().Msg("wait: refresher: checkSources complete")

	// finished probe
	<-run["probe"].Wait
	log.Trace().Msg("wait: probe: updated")

	<-run["refresher"].Wait
	log.Trace().Msg("wait: refresher: finished: failing")

	<-run["refresher"].Wait
	log.Trace().Msg("wait: refresher: finished: single")

	<-run["refresher"].Wait
	log.Trace().Msg("wait: refresher: tick")

	upcoming := ref.upcoming()
	stats := ref.stats.Snapshot()
	assert.Equal(t, 4, len(stats))
	for _, v := range upcoming {
		log.Trace().
			Int("source", v.Source).
			Stringer("in", v.Delay).
			Stringer("freq", v.Frequency).
			Msg("will refresh")
	}
}
