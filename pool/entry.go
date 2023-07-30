package pool

import (
	"context"
	"fmt"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/pool/counter"
)

// unexported unit test shim
var now = time.Now

type entry struct {
	Proxy          pmux.Proxy
	FirstSeen      int64
	LastSeen       int64
	ReanimateAfter time.Time
	Ok             bool
	Speed          time.Duration
	Timeouts       int
	Failures       int
	Offered        int
	Reanimated     int
	Succeed        int

	OfferShort   counter.RollingCounter
	SuccessShort counter.RollingCounter
	TimeoutShort counter.RollingCounter
	FailureShort counter.RollingCounter

	Offer1D   counter.RollingCounter
	Success1D counter.RollingCounter
	Timeout1D counter.RollingCounter
	Failure1D counter.RollingCounter

	HourOffered [24]int
	HourSucceed [24]int
}

func newEntry(proxy pmux.Proxy, speed time.Duration, evictSpanMinutes int16) *entry {
	now := now()
	return &entry{
		Ok:        true,
		Proxy:     proxy,
		Speed:     speed,
		FirstSeen: now.Unix(),
		LastSeen:  now.Unix(),

		OfferShort:   counter.NewRollingCounter(evictSpanMinutes, time.Minute),
		SuccessShort: counter.NewRollingCounter(evictSpanMinutes, time.Minute),
		TimeoutShort: counter.NewRollingCounter(evictSpanMinutes, time.Minute),
		FailureShort: counter.NewRollingCounter(evictSpanMinutes, time.Minute),

		Offer1D:   counter.NewRollingCounter(24, time.Hour),
		Success1D: counter.NewRollingCounter(24, time.Hour),
		Timeout1D: counter.NewRollingCounter(24, time.Hour),
		Failure1D: counter.NewRollingCounter(24, time.Hour),
	}
}

func (e *entry) MarkSuccess() {
	e.SuccessShort.Increment()
	e.Success1D.Increment()

	now := now()
	hour := now.Hour()
	if (e.Succeed + 1) < e.Offered {
		e.HourSucceed[hour]++
		e.Succeed++
	} else {
		// fix the potential mess
		e.HourSucceed[hour] = e.HourOffered[hour]
		e.Succeed = e.Offered
	}
	e.Ok = true
	e.LastSeen = now.Unix()
	// TODO: verify if prev request success will improve hitrate
	e.ReanimateAfter = time.Time{}
}

func (e *entry) MarkFailure(err error, shortTimeoutSleep time.Duration) {
	t, ok := err.(interface {
		Timeout() bool
	})
	e.Ok = false
	e.ReanimateAfter = now().Add(shortTimeoutSleep)
	if ok && t.Timeout() {
		e.TimeoutShort.Increment()
		e.Timeout1D.Increment()
		e.Timeouts++
	} else {
		e.FailureShort.Increment()
		e.Failure1D.Increment()
		e.Failures++
	}
}

func (e *entry) RequestsComplete() bool {
	offers := e.OfferShort.Sum()
	if offers == 0 {
		return false
	}
	return offers == e.RequestsShort()
}

func (e *entry) RequestsShort() int {
	return e.SuccessShort.Sum() + e.TimeoutShort.Sum() + e.FailureShort.Sum()
}

func (e *entry) SuccessRateShort() float64 {
	pass := float64(e.SuccessShort.Sum())
	fail := float64(e.RequestsShort())
	return pass / fail
}

func (e *entry) RequestsLong() int {
	return e.Success1D.Sum() + e.Timeout1D.Sum() + e.Failure1D.Sum()
}

func (e *entry) SuccessRateLong() float64 {
	pass := float64(e.Success1D.Sum())
	fail := float64(e.RequestsLong())
	return pass / fail
}

func (e *entry) Eviction(evictThresholdTimeouts, evictThresholdFailures, evictThresholdReanimations int, longTimeoutSleep time.Duration) bool {
	if !e.RequestsComplete() {
		return false
	}
	suceeded := e.SuccessShort.Sum()
	timedOut := e.TimeoutShort.Sum()
	failed := e.FailureShort.Sum()
	if suceeded == 0 && timedOut > evictThresholdTimeouts {
		e.Ok = false
		e.ReanimateAfter = now().Add(longTimeoutSleep)
		// don't evict just yet
		return false
	}
	if suceeded == 0 && failed > evictThresholdFailures {
		e.Ok = false
		return true
	}
	if suceeded == 0 && e.Reanimated > evictThresholdReanimations && (timedOut > 0 || failed > 0) {
		e.Ok = false
		return true
	}
	return false
}

// TODO: bug in offering: plenty of 551 errors, even though they should have been limited.
// perhaps we can do LastOffered and check it to be more than 3s (checker.Timeout) ago.
func (e *entry) ConsiderSkip(ctx context.Context, offerLimit int) bool {
	currentOffers := e.OfferShort.Sum()
	if currentOffers > offerLimit {
		// there were too many offers in the last 5 minutes
		return true
	}
	if e.SuccessShort.Sum() == 0 && e.FailureShort.Sum() == 3 {
		e.Ok = false
		e.ReanimateAfter = now().Add(30 * time.Second)
		return true
	}
	now := now()
	log := app.Log.From(ctx).
		With().
		Stringer("proxy", e.Proxy).
		Int("offered", e.Offered).
		Int("succeed", e.Succeed).
		Logger()
	if e.ReanimateAfter.After(now) {
		log.Trace().Str("until", e.DeadUntil()).Msg("dead")
		return true
	}
	if !e.ReanimateAfter.IsZero() {
		e.ReanimateAfter = time.Time{}
		e.Reanimated++
		log.Trace().Int("count", e.Reanimated).Msg("reanimated")
		e.Ok = true
	}
	if !e.Ok {
		// TODO: is this statement even reachable?
		log.Trace().Str("until", e.DeadUntil()).Msg("not ok")
		return true
	}
	e.HourOffered[now.Hour()]++
	e.Offered += 1
	e.OfferShort.Increment()
	e.Offer1D.Increment()
	log.Trace().Int("new_offered", e.Offered).Msg("offered")
	return false
}

func (e *entry) ReanimateIfNeeded() bool {
	if e.ReanimateAfter.IsZero() {
		return false
	}
	now := now()
	if e.ReanimateAfter.After(now) {
		return false
	}
	e.ReanimateAfter = time.Time{}
	e.Reanimated++
	e.Ok = true
	return true
}

func (e *entry) ForceReanimate() {
	e.ReanimateAfter = time.Time{}
	if !e.Ok {
		e.Reanimated++
		e.Ok = true
	}
}

func (e *entry) HourSuccessRate() [24]float32 {
	res := [24]float32{}
	for i := 0; i < 24; i++ {
		res[i] = 100 * float32(e.HourSucceed[i]) / float32(e.HourOffered[i])
	}
	return res
}

func (e *entry) SuccessRate() float32 {
	return 100 * float32(e.Succeed) / float32(e.Offered)
}

func (e *entry) TimeoutRate() float32 {
	return 100 * float32(e.Timeouts) / float32(e.Offered)
}

func (e *entry) SinceLastSeen() time.Duration {
	ls := time.Unix(e.LastSeen, 0)
	return time.Since(ls)
}

func (e *entry) LastSeenAgo() string {
	return e.SinceLastSeen().Round(time.Second).String()
}

func (e *entry) DeadUntil() string {
	if e.ReanimateAfter.IsZero() {
		return "-"
	}
	now := now()
	if now.After(e.ReanimateAfter) {
		return "-"
	}
	in := e.ReanimateAfter.Sub(now) * -1
	return in.Round(time.Second).String()
}

func (e *entry) String() string {
	speed := e.Speed.Round(time.Millisecond)
	ok := "+"
	if !e.Ok {
		ok = "-"
	}
	return fmt.Sprintf("%32s\t%s\t%s\t%3d%%\t%9s\t%5s\t%3d\t%3d\t%3d",
		e.Proxy, speed, ok, int(e.SuccessRate()),
		e.LastSeenAgo(), e.DeadUntil(), e.Offered, e.Succeed, e.Reanimated)
}
