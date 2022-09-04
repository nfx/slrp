package refresher

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/sources"

	"github.com/stretchr/testify/assert"
)

func TestNewRefresher(t *testing.T) {
	ref := NewRefresher(nil, nil, nil)
	assert.NotNil(t, ref)
	assert.Greater(t, len(ref.sources()), 1)
}

func TestUpcoming(t *testing.T) {
	ref := withStats(&Refresher{
		sources: func() []sources.Source {
			// we don't need any sources for this test
			return []sources.Source{}
		},
	})
	defer app.MockStart(ref)()

	_, err := ref.HttpGet(nil)
	assert.NoError(t, err)
}

func TestSnapshotForDashboard(t *testing.T) {
	ref := withStats(&Refresher{
		snapshot: make(chan chan plan),
		plan: plan{
			1: 10 * time.Second,
		},
		sources: func() []sources.Source {
			// we don't need any sources for this test
			return []sources.Source{}
		},
	})
	defer app.MockStart(ref)()

	plan := ref.Snapshot()
	assert.Equal(t, 10*time.Second, plan[1])
}

func TestSourcSomeFeeds(t *testing.T) {
	counter := counterProbe{}
	progress := make(chan progress)
	ref := withStats(&Refresher{
		probe:    counter,
		progress: progress,
		snapshot: make(chan chan plan),
		sources: func() []sources.Source {
			return []sources.Source{
				stubSource[0],
				stubSource[1],
			}
		},
	})
	defer app.MockStart(ref)()

	<-progress
	assert.Equal(t, 1, counter[2])
}

func TestProgressPicksUp(t *testing.T) {
	ref := withStats(&Refresher{
		progress: make(chan progress),
		pool:     nilPool{},
		snapshot: make(chan chan plan),
		sources: func() []sources.Source {
			return []sources.Source{}
		},
	})
	defer app.MockStart(ref)()

	ref.progress <- progress{
		ctx: context.Background(),
		Err: nil,
	}
}


func TestUpcomingDetails(t *testing.T) {
	ref := withStats(&Refresher{
		snapshot: make(chan chan plan),
		sources: func() []sources.Source {
			return []sources.Source{
				stubSource[0],
				stubSource[1],
			}
		},
	})
	upcoming := ref.upcoming()
	assert.Len(t, upcoming, 1)
	assert.Equal(t, time.Duration(0), upcoming[0].Delay)
}

func TestUpcomingDetailsSourceRunning(t *testing.T) {
	ref := withStats(&Refresher{
		snapshot: make(chan chan plan),
		sources: func() []sources.Source {
			return []sources.Source{
				stubSource[0],
				stubSource[1], // ID:2
			}
		},
	})
	ref.stats.Launch(2)
	upcoming := ref.upcoming()
	assert.Len(t, upcoming, 0)
}

func TestUpcomingDetailsSourceFailed(t *testing.T) {
	ref := withStats(&Refresher{
		snapshot: make(chan chan plan),
		sources: func() []sources.Source {
			return []sources.Source{
				stubSource[0],
				stubSource[1], // ID:2
			}
		},
	})
	ref.stats.Finish(2, fmt.Errorf("nope"))
	upcoming := ref.upcoming()
	assert.Len(t, upcoming, 1)
	assert.Equal(t, time.Duration(0), upcoming[0].Delay)
}

func TestUpcomingNewSourceAppeared(t *testing.T) {
	ref := withStats(&Refresher{
		snapshot: make(chan chan plan),
		sources: func() []sources.Source {
			return []sources.Source{
				stubSource[0],
				stubSource[1],
			}
		},
	})
	ref.sources = func() []sources.Source {
		return []sources.Source{
			stubSource[0],
			stubSource[1],
			stubSource[2],
		}
	}
	ref.stats.Finish(2, nil)
	upcoming := ref.upcoming()
	assert.Len(t, upcoming, 2)
	assert.Equal(t, time.Duration(0), upcoming[0].Delay)
}

func TestCheckSourcesUnrunSchedules(t *testing.T) {
	progress := make(chan progress)
	counter := counterProbe{}
	ref := withStats(&Refresher{
		probe:    counter,
		progress: progress,
		snapshot: make(chan chan plan),
		sources: func() []sources.Source {
			return []sources.Source{
				stubSource[0],
				stubSource[1],
			}
		},
	})
	trigger := time.Now()
	next := ref.checkSources(context.Background(), trigger)
	assert.Equal(t, trigger.Add(1*time.Minute), next)

	<-progress
	assert.Equal(t, 1, counter[2])
}

func TestCheckSourcesRunningWontSchedule(t *testing.T) {
	ref := withStats(&Refresher{
		snapshot: make(chan chan plan),
		sources: func() []sources.Source {
			return []sources.Source{
				stubSource[0],
				stubSource[1], // src:2
			}
		},
	})
	ref.stats.Launch(2)

	trigger := time.Now()
	next := ref.checkSources(context.Background(), trigger)
	assert.Equal(t, trigger.Add(1*time.Minute), next)
}

func TestCheckSourcesFailed(t *testing.T) {
	ref := withStats(&Refresher{
		snapshot: make(chan chan plan),
		plan:     plan{},
		sources: func() []sources.Source {
			return []sources.Source{
				stubSource[0],
				stubSource[1], // src:2
			}
		},
	})
	ref.stats.Finish(2, fmt.Errorf("nope"))

	trigger := time.Now()
	next := ref.checkSources(context.Background(), trigger)
	assert.Equal(t, trigger.Add(1*time.Minute), next)
}

func TestRefreshSessionSource(t *testing.T) {
	progress := make(chan progress)
	counter := counterProbe{}
	ref := withStats(&Refresher{
		probe:    counter,
		progress: progress,
		pool:     nilPool{},
		snapshot: make(chan chan plan),
		sources: func() []sources.Source {
			return []sources.Source{
				stubSource[2],
			}
		},
	})
	trigger := time.Now()
	ref.checkSources(context.Background(), trigger)

	<-progress
	assert.Equal(t, 1, counter[3])
}
