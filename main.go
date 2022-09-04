package main

import (
	"context"
	"embed"
	"flag"
	"fmt"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/checker"
	"github.com/nfx/slrp/history"
	"github.com/nfx/slrp/internal/updater"
	"github.com/nfx/slrp/ipinfo"
	"github.com/nfx/slrp/pool"
	"github.com/nfx/slrp/probe"
	"github.com/nfx/slrp/refresher"
	"github.com/nfx/slrp/serve"
	"github.com/nfx/slrp/stats"
)

var version = "devel"

//go:embed ui/build
var embedFrontend embed.FS

func main() {
	updatePtr := flag.Bool("update", false, "check for updates")
	flag.Parse()
	if *updatePtr {
		updater.AutoUpdate(version)
	}
	fmt.Printf("slrp v%s\n", version)
	app.Run(context.Background(), app.Factories{
		"ca":        serve.NewCA,
		"blacklist": probe.NewBlacklistApi,
		"checker":   checker.NewChecker,
		"dashboard": serve.NewDashboard,
		"history":   history.NewHistory,
		"ipinfo":    ipinfo.NewLookup,
		"mitm":      serve.NewMitmProxyServer,
		"pool":      pool.NewPool,
		"probe":     probe.NewProbe,
		"refresher": refresher.NewRefresher,
		"stats":     stats.NewStats,
		"ui":        app.MountSpaUI(embedFrontend),
	})
}
