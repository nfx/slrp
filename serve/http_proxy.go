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
	"net/url"
	"strings"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pmux"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type HttpProxyServer struct {
	http.Server
	listener  net.Listener
	transport http.RoundTripper
	signer    func(host string) (*tls.Certificate, error)
}

// defaultTransport ignores invalid TLS certs and has low timeouts
// only meant to be used for testing
var defaultTransport = &http.Transport{
	TLSClientConfig:       pmux.DefaultTlsConfig,
	DialContext:           pmux.DefaultDialer.DialContext,
	IdleConnTimeout:       15 * time.Second,
	TLSHandshakeTimeout:   5 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	MaxIdleConns:          100,
}

func NewTransparentProxy() *HttpProxyServer {
	ln, _ := net.Listen("tcp", "127.0.0.1:")
	srv := &HttpProxyServer{
		listener:  ln,
		transport: defaultTransport,
		signer:    defaultCA.Sign,
	}
	srv.Handler = srv
	return srv
}

// ListenAndServe uses listener configured in Start method, so that
// this type could be reused for both testing and real serving
func (srv *HttpProxyServer) ListenAndServe() error {
	if srv.listener == nil {
		return fmt.Errorf("listener is not configured")
	}
	log.Debug().Stringer("server", srv).Msg("started")
	return srv.Serve(srv.listener)
}

func (srv *HttpProxyServer) Proxy() pmux.Proxy {
	addr := srv.listener.Addr().String()
	return pmux.HttpProxy(addr)
}

func (srv *HttpProxyServer) String() string {
	return srv.Proxy().String()
}

func (srv *HttpProxyServer) Listen() (err error) {
	if srv.Handler == nil {
		srv.Handler = srv
	}
	srv.listener, err = net.Listen("tcp", srv.Addr)
	return err
}

func (srv *HttpProxyServer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if r.Method != "CONNECT" {
		srv.handleSimpleHttp(rw, r)
		return
	}
	srv.handleConnect(rw, r)
}

func (srv *HttpProxyServer) handleSimpleHttp(rw http.ResponseWriter, r *http.Request) {
	log := app.Log.From(r.Context()).With().Str("connection", "HTTP").Logger()
	log.Debug().Str("method", r.Method).Stringer("url", r.URL).Msg("init")
	res, err := srv.transport.RoundTrip(srv.rewrapRequest(log, r.URL.Scheme, r))
	if err != nil {
		log.Err(err).Msg("cannot do RoundTrip(r)")
		http.Error(rw, err.Error(), 470)
		return
	}
	if res.Body != nil {
		defer res.Body.Close()
	}
	for k, va := range res.Header {
		for _, v := range va {
			rw.Header().Add(k, v)
		}
	}
	rw.WriteHeader(res.StatusCode)
	if res.Body == nil {
		return
	}
	_, err = io.Copy(rw, res.Body)
	if err != nil {
		log.Err(err).Msg("cannot copy IO")
		return
	}
}

func (srv *HttpProxyServer) handleConnect(rw http.ResponseWriter, r *http.Request) {
	log := app.Log.From(r.Context()).With().
		Str("connection", "HTTPS").
		Stringer("server", srv.Proxy()).
		Logger()
	hijacker, ok := rw.(http.Hijacker)
	if !ok {
		rw.WriteHeader(501)
		return
	}
	rw.WriteHeader(200)
	src, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(rw, err.Error(), 471)
		return
	}
	log.Trace().Stringer("url", r.URL).Msg("hijacked")
	go func() {
		ctx := context.Background() // TODO: cleanup all requests?..
		ssl, err := srv.handleHandshake(ctx, src, r.URL.Host)
		if err != nil {
			log.Err(err).Msg("handshake failed")
			src.Close()
			return
		}
		defer ssl.Close()
		// TODO: buffer both reads and writes
		buf := bufio.NewReader(ssl)
		for {
			err := srv.handleInnerHttp(log, ssl, buf)
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				srv.writeError(ssl, 472, "Forwarding Failed", err)
				return
			}
		}
	}()
}

func (srv *HttpProxyServer) handleHandshake(ctx context.Context, src net.Conn, host string) (*tls.Conn, error) {
	cert, err := srv.signer(host)
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}
	ssl := tls.Server(src, &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{*cert},
	})
	return ssl, ssl.HandshakeContext(ctx)
}

func (srv *HttpProxyServer) rewrapRequest(log zerolog.Logger, scheme string, req *http.Request) *http.Request {
	return (&http.Request{
		Method: req.Method,
		Body:   req.Body,
		URL: &url.URL{
			Host:     req.Host,
			Path:     req.URL.Path,
			RawQuery: req.URL.RawQuery,
			Scheme:   scheme,
		},
		Header: req.Header,
		Proto:  req.Proto,
	}).WithContext(app.Log.To(req.Context(), log))
}

func (srv *HttpProxyServer) writeError(w *tls.Conn, httpCode int, status string, err error) {
	log.Warn().Err(err).Stringer("from", w.RemoteAddr()).Msg("forwarding failed")
	body := strings.NewReader(err.Error())
	(&http.Response{
		Body:       io.NopCloser(body),
		StatusCode: httpCode,
		Status:     status,
		ProtoMajor: 1,
		ProtoMinor: 1,
	}).Write(w)
}

func (srv *HttpProxyServer) handleInnerHttp(log zerolog.Logger, ssl *tls.Conn, buf *bufio.Reader) error {
	req, err := http.ReadRequest(buf)
	if err != nil {
		return fmt.Errorf("read request: %w", err)
	}
	defer req.Body.Close()
	res, err := srv.transport.RoundTrip(srv.rewrapRequest(log, "https", req))
	if err != nil {
		return fmt.Errorf("round trip: %w", err)
	}
	if res.Body != nil {
		defer res.Body.Close() // leak or not?..
	}
	return res.Write(ssl) // or chunked?..
}
