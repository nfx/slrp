package serve

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestHttpProxyServer_handleConnect_NoHijack(t *testing.T) {
	w := httptest.NewRecorder()

	(&HttpProxyServer{}).handleConnect(w, dummy)

	resp := w.Result()
	assert.Equal(t, 501, resp.StatusCode)
}

type hijackable struct {
	*httptest.ResponseRecorder
	*httptest.Server

	server    net.Conn
	client    net.Conn
	hijackErr error
}

func newHijackable() *hijackable {
	server, client := net.Pipe()
	srv := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(210)
		}))
	return &hijackable{
		ResponseRecorder: httptest.NewRecorder(),
		Server:           srv,
		server:           server,
		client:           client,
	}
}

func (w hijackable) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.hijackErr != nil {
		return nil, nil, w.hijackErr
	}
	conn, err := net.Dial("tcp", w.Listener.Addr().String())
	return conn, nil, err
}

func TestHttpProxyServer_handleConnect_CannotHijack(t *testing.T) {
	w := hijackable{
		ResponseRecorder: httptest.NewRecorder(),
		hijackErr:        fmt.Errorf("nope"),
	}

	(&HttpProxyServer{}).handleConnect(w, dummy)

	resp := w.Result()
	assert.Equal(t, 471, resp.StatusCode)
}

func TestHttpProxyServer_handleConnect_Hijacked(t *testing.T) {
	t.Skip("unfinished, pick up later")
	w := newHijackable()

	ca, _ := NewCA()
	(&HttpProxyServer{
		ca: ca,
	}).handleConnect(w, dummy)

	resp := w.Result()
	assert.Equal(t, 200, resp.StatusCode)

	var raw []byte
	_, err := w.server.Read(raw)
	assert.NoError(t, err)
}
