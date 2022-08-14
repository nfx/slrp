package serve

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/history"
	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/pool"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// listen mitm on a random port each time
	mitmDefaultAddr = "localhost:0"
}

func TestFlows(t *testing.T) {
	type proxy interface {
		Proxy() pmux.Proxy
	}
	type permutation struct {
		Name   string
		Target func(handler http.Handler) *httptest.Server
		Via    proxy
	}
	tests := []permutation{
		{
			Name:   "via HTTP to HTTP",
			Via:    NewTransparentProxy(),
			Target: httptest.NewServer,
		},
		{
			Name:   "via HTTP to HTTPS",
			Via:    NewTransparentProxy(),
			Target: httptest.NewTLSServer,
		},
		{
			Name:   "via HTTPS to HTTP",
			Via:    NewTransparentHttpsProxy(),
			Target: httptest.NewServer,
		},
		{
			Name:   "via HTTPS to HTTPS",
			Via:    NewTransparentHttpsProxy(),
			Target: httptest.NewTLSServer,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			srv := tt.Target(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(217)
				}))
			defer srv.Close()

			history := history.NewHistory()
			pool := pool.NewPool(history)
			mitm, runtime := app.MockStartSpin(
				NewMitmProxyServer(pool, *defaultCA),
				history, pool, tt.Via)
			defer runtime.Stop()

			pool.Add(runtime.Context(), tt.Via.Proxy(), 1*time.Second)
			assert.Equal(t, 1, pool.Len())

			log.Debug().
				Stringer("mitm", mitm).
				Stringer("proxy", tt.Via.Proxy()).
				Str("target", srv.URL).Msg("this request")

			req := mitm.Proxy().MustNewGetRequest(srv.URL)
			res, err := pmux.DefaultHttpClient.Do(req)
			require.NoError(t, err)

			assert.Equal(t, 217, res.StatusCode)
		})
	}
}

func TestMitm_HTTP_viaHTTP_toHTTP(t *testing.T) { // TODO: rename
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(217)
		}))
	defer srv.Close()

	transparentHttp := NewTransparentProxy()
	history := history.NewHistory()
	pool := pool.NewPool(history)
	mitm, runtime := app.MockStartSpin(
		NewMitmProxyServer(pool, *defaultCA),
		history, pool, transparentHttp)
	defer runtime.Stop()

	pool.Add(runtime.Context(), transparentHttp.Proxy(), 1*time.Second)
	assert.Equal(t, 1, pool.Len())

	log.Debug().
		Stringer("mitm", mitm).
		Stringer("proxy", transparentHttp).
		Str("target", srv.URL).Msg("this request")

	req := mitm.Proxy().MustNewGetRequest(srv.URL)
	res, err := pmux.DefaultHttpClient.Do(req)
	require.NoError(t, err)

	assert.Equal(t, 217, res.StatusCode)
}
