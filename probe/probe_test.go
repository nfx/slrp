package probe

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/history"
	"github.com/nfx/slrp/ipinfo"
	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/pool"
	"github.com/nfx/slrp/stats"
	"github.com/stretchr/testify/assert"
)

type failingChecker map[pmux.Proxy]error

func (m failingChecker) Check(_ context.Context, proxy pmux.Proxy) (time.Duration, error) {
	return 0, m[proxy]
}

type approvingChecker map[pmux.Proxy]time.Duration

func (m approvingChecker) Check(_ context.Context, proxy pmux.Proxy) (time.Duration, error) {
	return m[proxy], nil
}

func TestBasicProbe(t *testing.T) {
	firstProxy := pmux.HttpProxy("127.0.0.1:2345")
	secondProxy := pmux.HttpProxy("127.0.0.2:2345")

	checker := failingChecker{
		firstProxy:  context.DeadlineExceeded,
		secondProxy: fmt.Errorf("test failure"),
	}

	stats := stats.NewStats()
	history := history.NewHistory()
	pool := pool.NewPool(history, ipinfo.NoopIpInfo{
		Country: "Zimbabwe",
	})
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

	probe.Schedule(runtime.Context("probe"), firstProxy, 0)
	<-runtime["probe"].Wait

	probe.Schedule(runtime.Context("probe"), secondProxy, 0)
	<-runtime["probe"].Wait

	runtime["probe"].Spin()

	res, err := probe.HttpGet(&http.Request{
		URL: &url.URL{
			RawQuery: "filter=Offered:0",
		},
	})
	assert.NoError(t, err)

	s := res.(Stats2)
	assert.Equal(t, 1, s.Reverify)
	assert.Equal(t, 1, s.Blacklist)
}

func TestProbeMarshaling(t *testing.T) {
	secondProxy := pmux.HttpProxy("127.0.0.2:2345")

	checker := failingChecker{
		secondProxy: fmt.Errorf("test failure"),
	}

	stats := stats.NewStats()
	history := history.NewHistory()
	pool := pool.NewPool(history, ipinfo.NoopIpInfo{
		Country: "Zimbabwe",
	})
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
	<-runtime["probe"].Wait

	runtime["probe"].Spin()

	raw, err := probe.MarshalBinary()
	assert.NoError(t, err)

	loaded := NewProbe(stats, pool, checker)
	err = loaded.UnmarshalBinary(raw)
	assert.NoError(t, err)

	assert.Equal(t, 0, loaded.state.failuresInverted["test failure"])
}

func TestProbeDeleting(t *testing.T) {
	secondProxy := pmux.HttpProxy("127.0.0.2:2345")

	checker := approvingChecker{
		secondProxy: 1 * time.Second,
	}

	stats := stats.NewStats()
	history := history.NewHistory()
	pool := pool.NewPool(history, ipinfo.NoopIpInfo{
		Country: "Zimbabwe",
	})
	probe := NewProbe(stats, pool, checker)

	runtime := app.Singletons{
		"probe": probe,
		"hist":  history,
		"pool":  pool,
		"stats": stats,
	}.MockStart()
	defer runtime.Stop()
	runtime["stats"].Spin()
	runtime["pool"].Spin()

	probe.Schedule(runtime.Context("probe"), secondProxy, 0)

	<-runtime["probe"].Wait
	assert.Equal(t, 1, pool.Len())

	_, err := probe.HttpDeletetByID("http:127.0.0.2:2345", &http.Request{})
	assert.NoError(t, err)

	<-runtime["probe"].Wait
	assert.Equal(t, 0, pool.Len())
	assert.Equal(t, 0, probe.state.Blacklist[secondProxy])
	assert.Equal(t, "manual remove", probe.state.Failures[0])
}
