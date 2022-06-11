package serve

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pool"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type MitmProxyServer struct {
	pool     *pool.Pool
	ca       certWrapper
	sessions chan int
}

func NewMitmProxyServer(pool *pool.Pool, ca certWrapper) *MitmProxyServer {
	return &MitmProxyServer{
		pool:     pool,
		ca:       ca,
		sessions: make(chan int),
	}
}

func (mps *MitmProxyServer) Start(ctx app.Context) {
	go mps.counter(ctx)
	go mps.listenAndServe()
}

func (mps *MitmProxyServer) listenAndServe() {
	// TODO: make sure about private interfaces only!!!
	// TODO: stop the server on context stop
	(&http.Server{
		Addr:         "localhost:8090",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  15 * time.Second,
		Handler:      mps,
	}).ListenAndServe()
	log.Info().Str("listen", "http://localhost:9980").Msg("Started rotating proxy server")
}

func (mps *MitmProxyServer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	session := <-mps.sessions
	logger := log.With().Int("session", session).Logger()
	ctx := app.Log.To(r.Context(), logger)
	r = r.WithContext(ctx)
	if r.Method != "CONNECT" {
		mps.handleSimpleHttp(rw, r)
		return
	}
	mps.handleConnect(rw, r)
}

func (mps *MitmProxyServer) counter(ctx app.Context) {
	var start int
	for {
		start++
		select {
		case <-ctx.Done():
			return
		case mps.sessions <- start:
		}
	}
}

func (mps *MitmProxyServer) handleSimpleHttp(rw http.ResponseWriter, r *http.Request) {
	log := app.Log.From(r.Context()).With().Str("connection", "HTTP").Logger()
	log.Debug().Str("method", r.Method).Stringer("url", r.URL).Msg("init")
	r.RequestURI = "" // request uri cannot be set in client requests
	res, err := mps.pool.RoundTrip(r)
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

func (mps *MitmProxyServer) handleConnect(rw http.ResponseWriter, r *http.Request) {
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
		ssl, err := mps.handleHandshake(src, r.URL.Host)
		if err != nil {
			log.Err(err).Msg("handshake failed")
			return
		}
		defer ssl.Close()
		buf := bufio.NewReader(ssl)
		for {
			err := mps.handleInnerHttp(log, ssl, buf)
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

func (mps *MitmProxyServer) handleHandshake(src net.Conn, host string) (*tls.Conn, error) {
	cert, err := mps.ca.Sign(host)
	if err != nil {
		return nil, err
	}
	ssl := tls.Server(src, &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{*cert},
	})
	return ssl, ssl.Handshake()
}

func (mps *MitmProxyServer) handleInnerHttp(log zerolog.Logger, ssl *tls.Conn, buf *bufio.Reader) error {
	req, err := http.ReadRequest(buf)
	if err != nil {
		return err
	}
	defer req.Body.Close()
	res, err := mps.pool.RoundTrip((&http.Request{
		Method: req.Method,
		Body:   req.Body,
		URL: &url.URL{
			Host:     req.Host,
			Path:     req.URL.Path,
			RawQuery: req.URL.RawQuery,
			Scheme:   "https",
		},
		Header: req.Header,
		Proto:  req.Proto,
	}).WithContext(app.Log.To(req.Context(), log)))
	if err != nil {
		log.Err(err).Msg("round trip")
		return err
	}
	if res.Body != nil {
		defer res.Body.Close() // leak or not?..
	}
	return res.Write(ssl) // or chunked?..
}
