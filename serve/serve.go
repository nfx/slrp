package serve

import (
	"os"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/checker"
	"github.com/nfx/slrp/history"
	"github.com/nfx/slrp/pool"
	"github.com/nfx/slrp/probe"
	"github.com/nfx/slrp/refresher"
	"github.com/nfx/slrp/stats"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func NewServer() *app.Fabric {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, NoColor: false})
	// http.Handle("/", http.FileServer(http.Dir("../ui/build")))
	return &app.Fabric{
		State: "/tmp/harvester",
		Factories: app.Factories{
			"ca":        NewCA,
			"checker":   checker.NewChecker,
			"dashboard": NewDashboard,
			"history":   history.NewHistory,
			"mitm":      NewMitmProxyServer,
			"pool":      pool.NewPool,
			"probe":     probe.NewProbe,
			"refresher": refresher.NewRefresher,
			"stats":     stats.NewStats,
		},
	}
}
