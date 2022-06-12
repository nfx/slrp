package serve

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pool"

	"github.com/rs/zerolog/log"
)

type MitmProxyServer struct {
	HttpProxyServer
	sessions chan int
}

func NewMitmProxyServer(pool *pool.Pool, ca certWrapper) *MitmProxyServer {
	return &MitmProxyServer{
		HttpProxyServer: HttpProxyServer{
			transport: pool,
			ca:        ca,
		},
		sessions: make(chan int),
	}
}

func (mps *MitmProxyServer) Configure(c app.Config) error {
	// TODO: make sure about private interfaces only!!!
	mps.Addr = c.StrOr("addr", "localhost:8090")
	mps.ReadTimeout = c.DurOr("read_timeout", 15*time.Second)
	mps.IdleTimeout = c.DurOr("idle_timeout", 15*time.Second)
	mps.WriteTimeout = c.DurOr("write_timeout", 15*time.Second)
	mps.Handler = mps
	return mps.Listen()
}

func (mps *MitmProxyServer) transportProxy() func(*http.Request) (*url.URL, error) {
	return func(r *http.Request) (*url.URL, error) {
		if mps.Addr == "" {
			return nil, fmt.Errorf("mitm is not configured")
		}
		return &url.URL{
			Scheme: "http",
			Host:   mps.Addr,
		}, nil
	}
}

func (mps *MitmProxyServer) Start(ctx app.Context) {
	go mps.counter(ctx)
}

func (mps *MitmProxyServer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	session := <-mps.sessions
	logger := log.With().Int("session", session).Logger()
	ctx := app.Log.To(r.Context(), logger)
	r = r.WithContext(ctx)
	mps.HttpProxyServer.ServeHTTP(rw, r)
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
