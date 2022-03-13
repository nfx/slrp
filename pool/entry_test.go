package pool

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFailuresDelayReanimation(t *testing.T) {
	e := &entry{
		Offered: 10,
	}

	e.ConsiderSkip(context.Background())
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
	assert.True(t, e.ConsiderSkip(context.Background()))
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
	assert.False(t, e.ConsiderSkip(context.Background()))
	assert.Equal(t, 11, e.Offered)
	assert.Equal(t, 11, e.HourOffered[0])
	assert.Equal(t, 1, e.Reanimated)
	assert.Equal(t, time.Time{}, e.ReanimateAfter)
}

func TestConsiderSkip_TenOffersNoSuccessSkipsForLong(t *testing.T) {
	e := &entry{
		Offered:     10,
		HourOffered: i24a(0, 10),
		Ok:          true,
	}
	now = func() time.Time {
		return ti(0, 0, 0)
	}
	assert.True(t, e.ConsiderSkip(context.Background()))
	assert.False(t, e.Ok)
	assert.Equal(t, 10, e.Offered)
}
