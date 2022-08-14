package pmux

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.DurationFieldUnit = time.Second
}

func SetupHttpProxy(proxy *Proxy) func() {
	tproxy := httptest.NewServer(BypassProxy())
	*proxy = HttpProxy(strings.ReplaceAll(tproxy.URL, "http://", ""))
	return func() {
		tproxy.Close()
	}
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

func BypassProxy() http.Handler {
	return http.HandlerFunc(
		func(rw http.ResponseWriter, r *http.Request) {
			logt := log.With().Bool("test", true).Str("method", r.Method).Stringer("url", r.URL)
			if r.Method != "CONNECT" {
				log := logt.Str("type", "http").Logger()
				log.Info().Msg("start")
				res, err := http.DefaultTransport.RoundTrip(r)
				if err != nil {
					log.Err(err).Msg("failed")
					http.Error(rw, err.Error(), 470)
					return
				}
				defer res.Body.Close()
				for k, va := range res.Header {
					for _, v := range va {
						rw.Header().Add(k, v)
					}
				}
				rw.WriteHeader(res.StatusCode)
				_, err = io.Copy(rw, res.Body)
				if err != nil {
					log.Err(err).Msg("failed")
					return
				}
				return
			}
			// HTTPS
			log := logt.Str("type", "https").Logger()
			log.Info().Msg("start")
			hijacker, ok := rw.(http.Hijacker)
			if !ok {
				rw.WriteHeader(501)
				return
			}
			rw.WriteHeader(200)
			src, _, err := hijacker.Hijack()
			if err != nil {
				log.Err(err).Msg("failed")
				http.Error(rw, err.Error(), 471)
				return
			}
			dst, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
			if err != nil {
				log.Err(err).Msg("failed")
				http.Error(rw, err.Error(), 472)
				return
			}
			go transfer(dst, src)
			go transfer(src, dst)
		})
}
