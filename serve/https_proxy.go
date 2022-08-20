package serve

import (
	"crypto/tls"
	"fmt"
	"net"

	"github.com/nfx/slrp/pmux"
	"github.com/rs/zerolog/log"
)

type HttpsProxyServer struct {
	HttpProxyServer
	sslConfig *tls.Config
}

func NewTransparentHttpsProxy() *HttpsProxyServer {
	ln, _ := net.Listen("tcp", "127.0.0.1:")
	srv := &HttpsProxyServer{
		HttpProxyServer: HttpProxyServer{
			listener:  ln,
			transport: defaultTransport,
			signer:    defaultCA.Sign,
		},
		sslConfig: defaultCA.Config(),
	}
	srv.Handler = srv
	return srv
}

// ListenAndServe uses listener configured in Start method, so that
// this type could be reused for both testing and real serving
func (srv *HttpsProxyServer) ListenAndServe() error {
	if srv.listener == nil {
		return fmt.Errorf("listener is not configured")
	}
	log.Debug().Stringer("server", srv).Msg("started")
	tlsListener := tls.NewListener(srv.listener, srv.sslConfig)
	return srv.Serve(tlsListener)
}

func (srv *HttpsProxyServer) Proxy() pmux.Proxy {
	return pmux.HttpsProxy(srv.addr())
}

func (srv *HttpsProxyServer) String() string {
	return fmt.Sprintf("https://%s", srv.addr())
}
