package serve

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pmux"

	"github.com/rs/zerolog"
)

type HttpProxyServer struct {
	http.Server
	listener net.Listener

	transport http.RoundTripper
	ca        certWrapper
}

func NewLocalHttpProxy() *HttpProxyServer {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	srv := &HttpProxyServer{
		listener:  ln,
		transport: http.DefaultTransport,
	}
	srv.Handler = srv
	return srv
}

func NewLocalHttpsProxy() *HttpProxyServer {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	ca, err := NewCA()
	if err != nil {
		panic(err)
	}
	srv := &HttpProxyServer{
		listener:  ln,
		transport: http.DefaultTransport,
		ca:        ca,
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
	return srv.Serve(srv.listener)
}

func (srv *HttpProxyServer) Listen() (err error) {
	if srv.Handler == nil {
		srv.Handler = srv
	}
	srv.listener, err = net.Listen("tcp", srv.Addr)
	return err
}

func (srv *HttpProxyServer) Proxy() pmux.Proxy {
	return pmux.HttpProxy(srv.listener.Addr().String())
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
	log := app.Log.From(r.Context()).With().Str("connection", "HTTPS").Logger()
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
	// TODO: figure out context propagation
	go func() {
		ssl, err := srv.handleHandshake(src, r.URL.Host)
		if err != nil {
			log.Err(err).Msg("handshake failed")
			return
		}
		defer ssl.Close()
		buf := bufio.NewReader(ssl)
		for {
			err := srv.handleInnerHttp(log, ssl, buf)
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Err(err).Msg("request failed")
				return
			}
		}
	}()
}

func (srv *HttpProxyServer) handleHandshake(src net.Conn, host string) (*tls.Conn, error) {
	cert, err := srv.ca.Sign(host)
	if err != nil {
		return nil, err
	}
	ssl := tls.Server(src, &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{*cert},
	})
	return ssl, ssl.Handshake()
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

func (srv *HttpProxyServer) handleInnerHttp(log zerolog.Logger, ssl *tls.Conn, buf *bufio.Reader) error {
	req, err := http.ReadRequest(buf)
	if err != nil {
		return err
	}
	defer req.Body.Close()
	res, err := srv.transport.RoundTrip(srv.rewrapRequest(log, "https", req))
	if err != nil {
		log.Err(err).Msg("round trip")
		return err
	}
	if res.Body != nil {
		defer res.Body.Close() // leak or not?..
	}
	return res.Write(ssl) // or chunked?..
}
