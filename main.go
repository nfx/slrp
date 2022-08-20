package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/tj/go-update"
	"github.com/tj/go-update/progress"
	"github.com/tj/go-update/stores/github"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/checker"
	"github.com/nfx/slrp/history"
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
		m := &update.Manager{
			Command: "slrp",
			Store: &github.Store{
				Owner:   "nfx",
				Repo:    "slrp",
				Version: version,
			},
		}
		releases, err := m.LatestReleases()
		if err != nil {
			panic(err)
		}
		if len(releases) == 0 {
			println("no updates")
			return
		}
		asset := releases[0].FindTarball(runtime.GOOS, runtime.GOARCH)
		if asset == nil {
			fmt.Printf("no binary for %s %s\n", runtime.GOOS, runtime.GOARCH)
			return
		}
		println() // whitespace
		tarball, err := asset.DownloadProxy(progress.Reader)
		if err != nil {
			panic(err)
		}
		err = m.Install(tarball)
		if err != nil {
			panic(err)
		}
		os.Exit(0)
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
