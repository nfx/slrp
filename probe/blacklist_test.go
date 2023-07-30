package probe

import (
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/history"
	"github.com/nfx/slrp/ipinfo"
	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/pool"
	"github.com/nfx/slrp/ql/eval"
	"github.com/nfx/slrp/stats"
	"github.com/stretchr/testify/assert"
)

func TestBlacklist(t *testing.T) {
	secondProxy := pmux.HttpProxy("127.0.0.2:2345")

	checker := failingChecker{
		secondProxy: fmt.Errorf("test failure"),
	}

	stats := stats.NewStats()
	history := history.NewHistory()
	pool := pool.NewPool(history, ipinfo.NoopIpInfo{
		Country: "Zimbabwe",
	}, &net.Dialer{})
	probe := NewProbe(stats, pool, checker)

	runtime := app.Singletons{
		"probe": probe,
		"hist":  history,
		"pool":  pool,
		"stats": stats,
	}.MockStart()
	defer runtime.Stop()
	runtime["pool"].Spin()
	runtime["stats"].Spin()

	probe.Schedule(runtime.Context("probe"), secondProxy, 0)
	runtime["probe"].WaitAndSpin()

	b := NewBlacklistApi(probe, ipinfo.NewLookup())
	res, err := b.HttpGet(&http.Request{})
	assert.NoError(t, err)

	br := res.(*eval.QueryResult[blacklisted])
	assert.Len(t, br.Records, 1)
}
