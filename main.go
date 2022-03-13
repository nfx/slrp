package main

import (
	"context"
	"embed"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/checker"
	"github.com/nfx/slrp/history"
	"github.com/nfx/slrp/pool"
	"github.com/nfx/slrp/probe"
	"github.com/nfx/slrp/refresher"
	"github.com/nfx/slrp/serve"
	"github.com/nfx/slrp/stats"
)

//go:embed ui/build
var embedFrontend embed.FS

func main() {
	// go tool pprof -http=:8080 main http://127.0.0.1:8089/debug/pprof/profile
	app.Run(context.Background(), app.Factories{
		"ca":        serve.NewCA,
		"checker":   checker.NewChecker,
		"dashboard": serve.NewDashboard,
		"history":   history.NewHistory,
		"mitm":      serve.NewMitmProxyServer,
		"pool":      pool.NewPool,
		"probe":     probe.NewProbe,
		"refresher": refresher.NewRefresher,
		"stats":     stats.NewStats,
		"ui":        app.MountSpaUI(embedFrontend),
	})
}
