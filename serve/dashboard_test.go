package serve

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/history"
	"github.com/nfx/slrp/ipinfo"
	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/pool"
	"github.com/nfx/slrp/probe"
	"github.com/nfx/slrp/refresher"
	"github.com/nfx/slrp/sources"
	"github.com/nfx/slrp/stats"
	"github.com/stretchr/testify/assert"
)

type mockChecker map[pmux.Proxy]error

func (m mockChecker) Check(_ context.Context, proxy pmux.Proxy) (time.Duration, error) {
	return 0, m[proxy]
}

func TestDashboardRenders(t *testing.T) {
	ctx := context.Background()
	sources.Sources = []sources.Source{
		{
			ID:        1,
			Frequency: 1 * time.Hour,
			Seed:      true,
		},
	}

	firstProxy := pmux.HttpProxy("127.0.0.1:2345")
	secondProxy := pmux.HttpProxy("127.0.0.2:2345")

	checker := mockChecker{
		firstProxy:  context.DeadlineExceeded,
		secondProxy: fmt.Errorf("test failure"),
	}

	stats := stats.NewStats()
	history := history.NewHistory()
	pool := pool.NewPool(history, ipinfo.NoopIpInfo{
		Country: "Zimbabwe",
	})
	probe := probe.NewProbe(stats, pool, checker)
	refresher := refresher.NewRefresher(stats, pool, probe)
	dashboard := NewDashboard(refresher, probe, stats)

	runtime := app.Singletons{
		"stats":     stats,
		"probe":     probe,
		"refresher": refresher,
		"pool":      pool,
		"history":   history,
	}.MockStart()
	defer runtime.Stop()

	runtime["refresher"].WaitAndSpin()
	runtime["history"].Spin()
	runtime["stats"].Spin()
	runtime["pool"].Spin()

	stats.Launch(0) // reverify

	probe.Schedule(ctx, firstProxy, 1)
	<-runtime["probe"].Wait
	probe.Schedule(ctx, secondProxy, 1)
	<-runtime["probe"].Wait
	runtime["probe"].Spin()

	res, err := dashboard.HttpGet(nil)
	assert.NoError(t, err)
	d := res.(Dashboard)

	assert.Equal(t, "running", d.Refresh[0].State)
	assert.Equal(t, 1, d.Refresh[0].Blacklisted)
	assert.Equal(t, 1, d.Refresh[0].Timeouts)
}
