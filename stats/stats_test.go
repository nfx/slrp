package stats

import (
	"fmt"
	"testing"

	"github.com/nfx/slrp/app"

	"github.com/stretchr/testify/assert"
)

func TestPropagatesStatsCorrectly(t *testing.T) {
	s := NewStats()
	defer app.MockStart(s)()

	s.Launch(0)
	s.Update(0, Scheduled)
	s.Update(0, Ignored)
	assert.True(t, s.Snapshot().IsRunning(0))
	assert.False(t, s.Snapshot().IsRunning(34))

	s.Update(0, Scheduled)
	s.Update(0, New)
	s.Update(0, Probing)
	s.Update(0, Found)

	s.Update(0, Scheduled)
	s.Update(0, New)
	s.Update(0, Probing)
	s.Update(0, Timeout)

	s.Update(0, Scheduled)
	s.Update(0, New)
	s.Update(0, Probing)
	s.Update(0, Blacklisted)

	s.Finish(0, nil)
	assert.False(t, s.Snapshot().IsRunning(0))

	snapshot := s.Snapshot()

	assert.Equal(t, 0, snapshot[0].Scheduled)
	assert.Equal(t, 0, snapshot[0].New)
	assert.Equal(t, 0, snapshot[0].Probing)
	assert.Equal(t, 1, snapshot[0].Ignored)
	assert.Equal(t, 1, snapshot[0].Found)
	assert.Equal(t, 1, snapshot[0].Timeouts)
	assert.Equal(t, 1, snapshot[0].Blacklisted)
	assert.Equal(t, Idle, snapshot[0].State)
}

func TestLaunchingResetsCounters(t *testing.T) {
	s := NewStats()
	defer app.MockStart(s)()

	// we were doing some stuff peacefully...
	s.Update(0, Found)
	s.Update(0, Found)
	s.Update(0, Found)
	s.Update(0, Blacklisted)
	s.Update(0, Ignored)

	snapshot := s.Snapshot()
	assert.Equal(t, 3, snapshot[0].Found)
	assert.Equal(t, 1, snapshot[0].Blacklisted)
	assert.Equal(t, 1, snapshot[0].Ignored)
	assert.Equal(t, 0, snapshot[0].Progress)

	// but suddenly decided to start
	s.Launch(0)

	// which resets the counters
	snapshot = s.Snapshot()
	assert.Equal(t, 5, snapshot[0].anticipated)
	assert.Equal(t, 0, snapshot[0].Found)
	assert.Equal(t, 0, snapshot[0].Blacklisted)
	assert.Equal(t, 0, snapshot[0].Ignored)
	assert.Equal(t, 0, snapshot[0].Progress)
	assert.Equal(t, Running, snapshot[0].State)
}

func TestProgress(t *testing.T) {
	s := NewStats()
	defer app.MockStart(s)()

	s.LaunchAnticipated(0, 100)
	s.Update(0, Found)
	s.Update(0, Found)
	s.Update(0, Found)
	s.Update(0, Found)

	snapshot := s.Snapshot()
	assert.Equal(t, 4, snapshot[0].Progress)
	assert.Equal(t, Running, snapshot[0].State)

	s.LaunchAnticipated(0, 2)
	s.Update(0, Found)
	s.Update(0, Found)
	s.Update(0, Found)
	s.Update(0, Found)

	snapshot = s.Snapshot()
	assert.Equal(t, 100, snapshot[0].Progress)
}

func TestMarshalling(t *testing.T) {
	mctx := app.MockCtx()
	defer mctx.Cancel()
	s := NewStats()
	mctx.Start(s)

	failed := 1
	s.Update(failed, Found)
	s.Finish(failed, fmt.Errorf("just failed"))

	unfinished := 2
	s.Update(unfinished, Found)
	s.Update(unfinished, Blacklisted)

	// things won't get finished without full pipeline
	s.Update(0, Scheduled)
	s.Update(0, New)
	s.Update(0, Probing)
	s.Update(0, Found)
	s.Update(0, Finished)

	snapshot := s.Snapshot()
	assert.Equal(t, 3, len(snapshot))
	assert.Equal(t, Failed, snapshot[failed].State)
	assert.Equal(t, "just failed", snapshot[failed].Failure)
	assert.Equal(t, snapshot[0].Updated, snapshot.LastUpdated())

	b, err := s.MarshalBinary()
	assert.NoError(t, err)

	s2 := NewStats()
	err = s2.UnmarshalBinary(b)
	assert.NoError(t, err)
	mctx.Start(s2)

	snapshot = s2.Snapshot()
	assert.Equal(t, 1, len(snapshot))
}

func TestWrongUnmarshall(t *testing.T) {
	s := NewStats()
	err := s.UnmarshalBinary([]byte{1})
	assert.EqualError(t, err, "unexpected EOF")
}
