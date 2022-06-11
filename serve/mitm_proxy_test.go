package serve

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/history"
	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/pool"
	"github.com/stretchr/testify/assert"
)

func TestMitmWorks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var proxy pmux.Proxy
	defer pmux.SetupHttpProxy(&proxy)()

	ca, err := NewCA()
	assert.NoError(t, err)

	history := history.NewHistory()
	pool := pool.NewPool(history)
	mitm := NewMitmProxyServer(pool, ca)

	runtime := app.Singletons{
		"pool":    pool,
		"mitm":    mitm,
		"history": history,
	}.MockStart()
	defer runtime.Stop()
	runtime["pool"].Spin()
	runtime["mitm"].Spin()
	runtime["history"].Spin()

	pool.Add(ctx, proxy, 1*time.Second)
	assert.Equal(t, 1, pool.Len())

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(&url.URL{
				Scheme: "http",
				Host:   "localhost:8090",
			}),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	// TODO: spin up test servers not to get to internet for no reason
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://httpbin.org/get", nil)
	res, err := client.Do(req)
	assert.NoError(t, err)

	// TODO: make it working properly
	assert.Equal(t, 429, res.StatusCode)
}
