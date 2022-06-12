package serve

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/history"
	"github.com/nfx/slrp/pool"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMitmWorksForHttp(t *testing.T) {
	ca, err := NewCA()
	assert.NoError(t, err)

	proxy := NewLocalHttpProxy()
	history := history.NewHistory()
	pool := pool.NewPool(history)
	mitm, runtime := app.MockStartSpin(NewMitmProxyServer(pool, ca), history, pool, proxy)
	defer runtime.Stop()

	pool.Add(runtime.Context(), proxy.Proxy(), 1*time.Second)
	assert.Equal(t, 1, pool.Len())

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: mitm.transportProxy(),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	// TODO: spin up test servers not to get to internet for no reason
	req, _ := http.NewRequestWithContext(runtime.Context(), "GET", "http://httpbin.org/get", nil)
	res, err := client.Do(req)
	require.NoError(t, err)

	// TODO: make it working properly
	assert.Equal(t, 200, res.StatusCode)
}

func TestMitmWorksForHttps(t *testing.T) {
	ca, err := NewCA()
	assert.NoError(t, err)

	proxy := NewLocalHttpsProxy()
	history := history.NewHistory()
	pool := pool.NewPool(history)
	mitm, runtime := app.MockStartSpin(NewMitmProxyServer(pool, ca), history, pool, proxy)
	defer runtime.Stop()

	pool.Add(runtime.Context(), proxy.Proxy(), 1*time.Second)
	assert.Equal(t, 1, pool.Len())

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: mitm.transportProxy(),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	// TODO: spin up test servers not to get to internet for no reason
	req, _ := http.NewRequestWithContext(runtime.Context(), "GET", "https://httpbin.org/get", nil)
	res, err := client.Do(req)
	require.NoError(t, err)

	// TODO: make it working properly
	assert.Equal(t, 200, res.StatusCode)
}
