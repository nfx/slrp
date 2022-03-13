package pool

import (
	"context"
	"fmt"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pmux"
)

// unexported unit test shim
var now = time.Now

type entry struct { // todo: package private
	Proxy          pmux.Proxy
	FirstSeen      int64
	LastSeen       int64
	ReanimateAfter time.Time
	Ok             bool
	Speed          time.Duration
	Seen           int
	Timeouts       int
	Offered        int
	Reanimated     int
	Succeed        int
	HourOffered    [24]int
	HourSucceed    [24]int
}

func (e *entry) MarkSeen() {
	e.LastSeen = now().Unix()
	e.Seen++
}

func (e *entry) MarkSuccess() {
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
}

func (e *entry) MarkFailure(err error) {
	t, ok := err.(interface {
		Timeout() bool
	})
	e.Ok = false
	e.ReanimateAfter = now().Add(5 * time.Minute)
	if ok && t.Timeout() {
		e.Timeouts++
	}
}

func (e *entry) ConsiderSkip(ctx context.Context) bool {
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
	if e.HourOffered[now.Hour()] > 3 && e.HourSucceed[now.Hour()] == 0 {
		e.Ok = false
		e.ReanimateAfter = time.Date(
			now.Year(), now.Month(), now.Day(),
			now.Hour()+1, 5, 0, 0, now.Location())
		log.Trace().
			Str("until", e.DeadUntil()).
			Int("hour", now.Hour()).
			Int("offered", e.HourOffered[now.Hour()]).
			Int("succeeded", e.HourSucceed[now.Hour()]).
			Msg("skipping for an hour")
		return true
	}
	if e.Timeouts > 12 && e.Succeed == 0 {
		e.Ok = false // TODO: outer process
		log.Trace().Int("timeouts", e.Timeouts).Msg("to be blacklisted")
		return true
	}
	e.HourOffered[now.Hour()]++
	e.Offered += 1
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
