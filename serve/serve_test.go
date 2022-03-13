package serve

import (
	"context"
	"testing"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/checker"
	"github.com/nfx/slrp/history"
	"github.com/nfx/slrp/pool"
	"github.com/nfx/slrp/probe"
	"github.com/nfx/slrp/refresher"
	"github.com/nfx/slrp/stats"
)

func TestServe(t *testing.T) {
	// run with race detector
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app.Run(ctx, app.Factories{
		"ca":        NewCA,
		"dashboard": NewDashboard,
		"mitm":      NewMitmProxyServer,
		"checker":   checker.NewChecker,
		"history":   history.NewHistory,
		"pool":      pool.NewPool,
		"probe":     probe.NewProbe,
		"refresher": refresher.NewRefresher,
		"stats":     stats.NewStats,
	})
}
