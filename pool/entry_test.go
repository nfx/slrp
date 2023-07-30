package pool

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testPoolCfg = &monitorConfig{
	shards:                     1,
	offerLimit:                 25,
	evictSpanMinutes:           5,
	shortTimeoutSleep:          1 * time.Minute,
	longTimeoutSleep:           1 * time.Hour,
	evictThresholdTimeouts:     3,
	evictThresholdFailures:     3,
	evictThresholdReanimations: 10,
}

func TestEntryMarkFailure(t *testing.T) {
	e := &entry{Ok: true}
	now = func() time.Time {
		return ti(0, 2, 0)
	}

	e.MarkFailure(context.DeadlineExceeded, 1*time.Minute)

	assert.Equal(t, false, e.Ok)
	assert.Equal(t, 1, e.Timeouts)
	assert.Equal(t, ti(0, 3, 0), e.ReanimateAfter)
}

func TestEntryReanimateIfNeeded_Zero(t *testing.T) {
	e := &entry{Ok: true}

	res := e.ReanimateIfNeeded()

	assert.Equal(t, false, res)
}

func TestEntryReanimateIfNeeded_ReanimateAfterAfterNow(t *testing.T) {
	e := &entry{ReanimateAfter: ti(0, 3, 0)}
	now = func() time.Time {
		return ti(0, 2, 0)
	}

	res := e.ReanimateIfNeeded()

	assert.Equal(t, false, res)
}

func TestEntryReanimateIfNeeded_ReanimateAfterBeforeNow(t *testing.T) {
	e := &entry{ReanimateAfter: ti(0, 1, 0)}
	now = func() time.Time {
		return ti(0, 2, 0)
	}

	res := e.ReanimateIfNeeded()

	assert.Equal(t, true, res)
	assert.Equal(t, 1, e.Reanimated)
	assert.Equal(t, true, e.Ok)
	assert.Equal(t, time.Time{}, e.ReanimateAfter)
}

func TestEntryForceReanimate(t *testing.T) {
	e := &entry{Ok: false}

	e.ForceReanimate()

	assert.Equal(t, true, e.Ok)
	assert.Equal(t, 1, e.Reanimated)
}

func TestEntryStringerBasic(t *testing.T) {
	e := &entry{Succeed: 3, Offered: 10, LastSeen: now().Unix()}
	res := e.String()
	assert.NotEmpty(t, res)
}

func TestFailuresDelayReanimation(t *testing.T) {
	e := &entry{
		Offered: 10,
	}

	e.ConsiderSkip(context.Background(), 1000)
}

func ti(h, m, s int) time.Time {
	return time.Date(2022, 1, 17, h, m, s, 0, time.Local)
}

func TestConsiderSkip_Dead(t *testing.T) {
	e := &entry{
		Offered:        10,
		ReanimateAfter: ti(0, 15, 0),
	}
	now = func() time.Time {
		return ti(0, 0, 0)
	}
	assert.True(t, e.ConsiderSkip(context.Background(), 100))
	assert.False(t, e.Ok)
	assert.Equal(t, 10, e.Offered)
}

func i24a(offset, value int) (res [24]int) {
	res[offset] = value
	return res
}

func TestConsiderSkip_Reanimate(t *testing.T) {
	e := &entry{
		Offered:        10,
		ReanimateAfter: ti(0, 1, 0),
		HourOffered:    i24a(0, 10),
		HourSucceed:    i24a(0, 5),
		Ok:             true,
	}
	now = func() time.Time {
		return ti(0, 2, 0)
	}
	assert.True(t, e.Ok)
	assert.False(t, e.ConsiderSkip(context.Background(), 100))
	assert.Equal(t, 11, e.Offered)
	assert.Equal(t, 11, e.HourOffered[0])
	assert.Equal(t, 1, e.Reanimated)
	assert.Equal(t, time.Time{}, e.ReanimateAfter)
}
