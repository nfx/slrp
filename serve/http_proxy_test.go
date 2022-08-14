package serve

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/nfx/slrp/internal/qa"
	"github.com/nfx/slrp/pmux"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebugTransparentProxy(t *testing.T) {
	qa.RunOnlyInDebug(t)
	httpProxy := NewTransparentProxy()
	defer httpProxy.Close()
	curlTmpl := "curl --verbose --proxy-insecure --proxy %s --insecure %s"
	log.Info().Msgf(curlTmpl, httpProxy, "http://httpbin.org/get")
	log.Info().Msgf(curlTmpl, httpProxy, "https://httpbin.org/get")
	httpProxy.ListenAndServe()
}

func TestNewTransparentProxy_HTTP_to_HTTPS(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(217)
	}))
	defer srv.Close()

	httpProxy := NewTransparentProxy()
	go httpProxy.ListenAndServe()
	defer httpProxy.Close()

	log.Debug().
		Stringer("proxy", httpProxy).
		Str("target", srv.URL).
		Msg("this request")

	req := httpProxy.Proxy().MustNewGetRequest(srv.URL)
	res, err := pmux.DefaultHttpClient.Do(req)
	require.NoError(t, err)

	assert.Equal(t, 217, res.StatusCode)
}

func TestNewTransparentProxy_HTTP_to_HTTPS_sing_failed(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(217)
	}))
	defer srv.Close()

	httpProxy := NewTransparentProxy()
	httpProxy.signer = func(host string) (*tls.Certificate, error) {
		return nil, fmt.Errorf("nope")
	}
	go httpProxy.ListenAndServe()
	defer httpProxy.Close()

	log.Debug().
		Stringer("proxy", httpProxy).
		Str("target", srv.URL).
		Msg("this request")

	req := httpProxy.Proxy().MustNewGetRequest(srv.URL)
	_, err := pmux.DefaultHttpClient.Do(req)
	assert.True(t, errors.Is(err, io.EOF))
}

func TestNewTransparentProxy_HTTP_to_HTTPS_ConnectionClosed(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hijacker, _ := w.(http.Hijacker)
		conn, _, _ := hijacker.Hijack()
		conn.Close()
	}))
	defer srv.Close()

	httpProxy := NewTransparentProxy()
	go httpProxy.ListenAndServe()
	defer httpProxy.Close()

	log.Debug().
		Stringer("proxy", httpProxy).
		Str("target", srv.URL).
		Msg("this request")

	req := httpProxy.Proxy().MustNewGetRequest(srv.URL)
	_, err := pmux.DefaultHttpClient.Do(req)
	assert.True(t, errors.Is(err, io.EOF))
}

func TestNewTransparentProxy_HTTP_to_HTTPS_InvalidRequest(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(501) // won't reach...
	}))
	defer srv.Close()

	httpProxy := NewTransparentProxy()
	go httpProxy.ListenAndServe()
	defer httpProxy.Close()

	log.Debug().
		Stringer("proxy", httpProxy).
		Str("target", srv.URL).
		Msg("this request")

	ctx := context.Background()
	conn, err := pmux.DefaultDialer.DialContext(ctx, "tcp", httpProxy.listener.Addr().String())
	require.NoError(t, err)

	_, err = conn.Write([]byte(fmt.Sprintf("CONNECT %s HTTP/1.1\n\n", srv.Listener.Addr())))
	require.NoError(t, err)

	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)

	ssl := tls.Client(conn, pmux.DefaultTlsConfig)
	defer ssl.Close()

	err = ssl.HandshakeContext(ctx)
	require.NoError(t, err)

	_, err = ssl.Write([]byte("Harmless request ...\n"))
	require.NoError(t, err)

	reader = bufio.NewReader(ssl)
	response, err = http.ReadResponse(reader, nil)
	require.NoError(t, err)
	assert.Equal(t, 472, response.StatusCode)

	body, err := io.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.Equal(t, `read request: malformed HTTP version "..."`, string(body))
}

func TestNewTransparentProxy_HTTP_to_HTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(217)
	}))
	defer srv.Close()

	httpProxy := NewTransparentProxy()
	go httpProxy.ListenAndServe()
	defer httpProxy.Close()

	log.Debug().
		Stringer("proxy", httpProxy).
		Str("target", srv.URL).
		Msg("this request")

	req := httpProxy.Proxy().MustNewGetRequest(srv.URL)
	res, err := pmux.DefaultHttpClient.Do(req)
	require.NoError(t, err)

	assert.Equal(t, 217, res.StatusCode)
}

func TestHttpProxyServer_ListenAndServe_NoConf(t *testing.T) {
	err := (&HttpProxyServer{}).ListenAndServe()
	assert.EqualError(t, err, "listener is not configured")
}

type staticRT func(*http.Request) (*http.Response, error)

func (s staticRT) RoundTrip(r *http.Request) (*http.Response, error) {
	return s(r)
}

var dummy = &http.Request{
	URL: &url.URL{
		Scheme: "http",
		Host:   "localhost",
	},
}

func TestHttpProxyServer_handleSimpleHttp_RoundTripFailed(t *testing.T) {
	w := httptest.NewRecorder()

	(&HttpProxyServer{
		transport: staticRT(func(r *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("nope")
		}),
	}).handleSimpleHttp(w, dummy)

	resp := w.Result()
	raw, _ := io.ReadAll(resp.Body)

	assert.Equal(t, 470, resp.StatusCode)
	assert.Equal(t, "nope\n", string(raw))
}

func TestHttpProxyServer_handleSimpleHttp_ForwardEmptyBody(t *testing.T) {
	w := httptest.NewRecorder()

	(&HttpProxyServer{
		transport: staticRT(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 201,
				Header: http.Header{
					"X-Test": []string{"yes"},
				},
			}, nil
		}),
	}).handleSimpleHttp(w, dummy)

	resp := w.Result()
	raw, _ := io.ReadAll(resp.Body)

	// empty body is not nil, but zero-byte
	assert.Equal(t, "", string(raw))
	assert.Equal(t, 201, resp.StatusCode)
	assert.Equal(t, "yes", resp.Header.Get("X-Test"))
}

type failingReader string

func (f failingReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("%s", f)
}

func TestHttpProxyServer_handleSimpleHttp_FailingReader(t *testing.T) {
	w := httptest.NewRecorder()

	(&HttpProxyServer{
		transport: staticRT(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 201,

				// TODO: make some better reporting to client
				Body: io.NopCloser(failingReader("nope")),
			}, nil
		}),
	}).handleSimpleHttp(w, dummy)

	resp := w.Result()
	raw, _ := io.ReadAll(resp.Body)

	assert.Equal(t, "", string(raw))
	assert.Equal(t, 201, resp.StatusCode)
}

type failingListener string

func (f failingListener) Accept() (net.Conn, error) {
	return nil, fmt.Errorf("always: %s", f)
}

func (f failingListener) Close() error {
	return fmt.Errorf("always: %s", f)
}

func (f failingListener) Addr() net.Addr {
	return dummyAddr(f)
}

type dummyAddr string

func (d dummyAddr) Network() string {
	return "dummy"
}

// name of the network (for example, "tcp", "udp")
func (d dummyAddr) String() string {
	return string(d)
}

func TestHttpProxyServer_handleConnect_NoHijack(t *testing.T) {
	w := httptest.NewRecorder()
	(&HttpProxyServer{
		listener: failingListener(mitmDefaultAddr),
	}).handleConnect(w, dummy)

	resp := w.Result()
	assert.Equal(t, 501, resp.StatusCode)
}

type failingHijackable struct {
	*httptest.ResponseRecorder
	hijackErr error
}

func (w failingHijackable) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, w.hijackErr
}

func TestHttpProxyServer_handleConnect_CannotHijack(t *testing.T) {
	w := failingHijackable{
		ResponseRecorder: httptest.NewRecorder(),
		hijackErr:        fmt.Errorf("nope"),
	}

	(&HttpProxyServer{
		listener: failingListener(mitmDefaultAddr),
	}).handleConnect(w, dummy)

	resp := w.Result()
	assert.Equal(t, 200, resp.StatusCode)
}
