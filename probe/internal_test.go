package probe

import (
	"testing"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/stats"
	"github.com/stretchr/testify/assert"
)

func TestInternalScheduleEmptyIP(t *testing.T) {
	stats, runtime := app.MockStartSpin(stats.NewStats())
	defer runtime.Stop()

	internal := newInternal(stats, make(chan verify), 1)
	internal.handleScheduled(verify{
		ctx: runtime.Context(),
	})

	assert.Equal(t, 1, stats.Snapshot()[0].Ignored)
}

func TestInternalScheduleSeenOtherSource(t *testing.T) {
	stats, runtime := app.MockStartSpin(stats.NewStats())
	defer runtime.Stop()

	internal := newInternal(stats, make(chan verify), 1)

	proxy := pmux.HttpProxy("127.0.0.2:2345")
	internal.SeenSources[proxy] = map[int]bool{99: true}
	internal.Seen[proxy] = true
	internal.LastReverified[proxy] = reVerify{
		Proxy:   proxy,
		Attempt: 20,
		After:   100,
	}

	internal.handleScheduled(verify{
		ctx:     runtime.Context(),
		Proxy:   proxy,
		Source:  1,
		Attempt: 1,
	})

	assert.Zero(t, internal.LastReverified[proxy])
	assert.Len(t, internal.SeenSources[proxy], 2)
	assert.True(t, internal.SeenSources[proxy][1])
	assert.Equal(t, 1, stats.Snapshot()[1].Ignored)
}

func TestInternalScheduleWasBlacklisted(t *testing.T) {
	stats, runtime := app.MockStartSpin(stats.NewStats())
	defer runtime.Stop()

	internal := newInternal(stats, make(chan verify), 1)

	proxy := pmux.HttpProxy("127.0.0.2:2345")
	internal.Blacklist[proxy] = 0
	internal.LastReverified[proxy] = reVerify{
		Proxy:   proxy,
		Attempt: 20,
		After:   100,
	}

	internal.handleScheduled(verify{
		ctx:     runtime.Context(),
		Proxy:   proxy,
		Source:  1,
		Attempt: 1,
	})

	assert.Zero(t, internal.LastReverified[proxy])
	assert.Equal(t, 1, stats.Snapshot()[1].Ignored)
}

func TestInternalScheduleInReverify(t *testing.T) {
	stats, runtime := app.MockStartSpin(stats.NewStats())
	defer runtime.Stop()

	internal := newInternal(stats, make(chan verify), 1)

	proxy := pmux.HttpProxy("127.0.0.2:2345")
	internal.LastReverified[proxy] = reVerify{
		Proxy:   proxy,
		Attempt: 20,
		After:   100,
	}

	internal.handleScheduled(verify{
		ctx:    runtime.Context(),
		Proxy:  proxy,
		Source: 1,
	})

	assert.Equal(t, 1, stats.Snapshot()[1].Ignored)
}

func TestInternalScheduleHandleReverify(t *testing.T) {
	stats, runtime := app.MockStartSpin(stats.NewStats())
	defer runtime.Stop()

	internal := newInternal(stats, make(chan verify), 1)

	proxy := pmux.Socks4Proxy("127.0.0.2:2345")
	internal.LastReverified[proxy] = reVerify{
		Proxy:   proxy,
		Attempt: 20,
	}

	proxy2 := pmux.Socks5Proxy("127.0.0.3:2345")

	internal.LastReverified[proxy2] = reVerify{
		Proxy:   proxy,
		Attempt: 2,
	}

	internal.handleReverify(runtime.Context())

	scheduled := <-internal.scheduled
	assert.Equal(t, proxy, scheduled.Proxy)

	assert.Equal(t, "exceeded 5 reverifies", internal.Failures[0])
	assert.Equal(t, 0, internal.Blacklist[proxy])
}
