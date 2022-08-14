package serve

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nfx/slrp/internal/qa"
	"github.com/nfx/slrp/pmux"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebugTransparentHttpsProxy(t *testing.T) {
	qa.RunOnlyInDebug(t)
	httpsProxy := NewTransparentHttpsProxy()
	defer httpsProxy.Close()
	curlTmpl := "curl --verbose --proxy-insecure --proxy %s --insecure %s"
	log.Info().Msgf(curlTmpl, httpsProxy, "http://httpbin.org/get")
	log.Info().Msgf(curlTmpl, httpsProxy, "https://httpbin.org/get")
	httpsProxy.ListenAndServe()
}

func TestNewTransparentHttpsProxy_HTTPS_to_HTTPS(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(217)
	}))
	defer srv.Close()

	httpsProxy := NewTransparentHttpsProxy()
	go httpsProxy.ListenAndServe()
	defer httpsProxy.Close()

	log.Debug().
		Stringer("proxy", httpsProxy).
		Str("target", srv.URL).
		Msg("this request")

	req := httpsProxy.Proxy().MustNewGetRequest(srv.URL)
	res, err := pmux.DefaultHttpClient.Do(req)
	require.NoError(t, err)

	assert.Equal(t, 217, res.StatusCode)
}

func TestNewTransparentHttpsProxy_HTTPS_to_HTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(217)
	}))
	defer srv.Close()

	httpsProxy := NewTransparentHttpsProxy()
	go httpsProxy.ListenAndServe()
	defer httpsProxy.Close()

	log.Debug().
		Stringer("proxy", httpsProxy).
		Str("target", srv.URL).
		Msg("this request")

	req := httpsProxy.Proxy().MustNewGetRequest(srv.URL)
	res, err := pmux.DefaultHttpClient.Do(req)
	require.NoError(t, err)

	assert.Equal(t, 217, res.StatusCode)
}
