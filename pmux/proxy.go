package pmux

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/net/proxy"
)

type proto int8

const (
	// don't change
	HTTP proto = iota
	HTTPS
	SOCKS4
	SOCKS5
)

var protoMap = map[string]proto{
	"http":   HTTP,
	"https":  HTTPS,
	"socks4": SOCKS4,
	"socks5": SOCKS5,
}

var reverseProtoMap = map[proto]string{
	HTTP:   "http",
	HTTPS:  "https",
	SOCKS4: "socks4",
	SOCKS5: "socks5",
}

type Proxy struct {
	IP    net.IP
	Port  uint16
	Proto proto
}

func (p *Proxy) URL() *url.URL {
	return &url.URL{
		Host:   fmt.Sprintf("%s:%d", p.IP, p.Port),
		Scheme: reverseProtoMap[p.Proto],
	}
}

func (p Proxy) String() string {
	// use custom striner instead of URL stringer for performance
	return fmt.Sprintf("%s://%s:%d", reverseProtoMap[p.Proto], p.IP, p.Port)
}

func (p *Proxy) Equal(o Proxy) bool {
	if len(p.IP) == 0 || len(o.IP) == 0 {
		return false
	}
	// if !p.IP.Equal(p.IP) {
	a := p.IP[0] != o.IP[0]
	b := p.IP[1] != o.IP[1]
	c := p.IP[2] != o.IP[2]
	d := p.IP[3] != o.IP[3]
	if a || b || c || d {
		return false
	}
	if p.Port != o.Port {
		return false
	}
	if p.Proto != o.Proto {
		return false
	}
	return true
}

type ckey int

const ProxyURL ckey = iota

func (p *Proxy) InContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ProxyURL, &Proxy{
		// should be copy, otherwise a data race
		// sorting entry list & dialing a connection
		IP:    p.IP,
		Port:  p.Port,
		Proto: p.Proto,
	})
}

func (p *Proxy) Valid() bool {
	return p.Port != 0
}

func (p *Proxy) IsTunnel() bool {
	return p.Proto == SOCKS4 || p.Proto == SOCKS5
}

// returns int representation of IP address
func (p *Proxy) Uint32() uint32 {
	return binary.BigEndian.Uint32(p.IP)
}

func (p *Proxy) Bucket(buckets int) int {
	return int(p.Uint32()) % buckets
}

func GetProxyFromContext(ctx context.Context) *Proxy {
	p := ctx.Value(ProxyURL)
	if p == nil {
		return nil
	}
	proxy, ok := p.(*Proxy)
	if !ok {
		return nil
	}
	return proxy
}

// experiment with net.Dialer.Control to bypass TCP fingerprinting
// http://witch.valdikss.org.ru/
// https://en.wikipedia.org/wiki/TCP/IP_stack_fingerprinting
// https://stackoverflow.com/a/52426887/277035
func dialProxiedConnection(ctx context.Context, network, addr string) (net.Conn, error) {
	p := GetProxyFromContext(ctx)
	// given the wish and time to implement a proxy chain, it should be done here
	if p == nil || !p.IsTunnel() {
		// use normal connection establishment in one of two cases:
		// a) no proxy is specified
		// b) HTTP proxy (handled higher on the stack)
		return (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 0,
		}).DialContext(ctx, network, addr)
	}
	dialer, err := proxy.FromURL(p.URL(), proxy.Direct)
	if err != nil {
		return nil, err
	}
	return dialer.Dial(network, addr)
}

func pickHttpProxyFromContext(r *http.Request) (*url.URL, error) {
	p := GetProxyFromContext(r.Context())
	if p == nil {
		return nil, nil
	}
	if p.IsTunnel() {
		// handled in DialContext
		return nil, nil
	}
	return p.URL(), nil
}

func ContextualHttpTransport() *http.Transport {
	return &http.Transport{
		DialContext: dialProxiedConnection,
		Proxy:       pickHttpProxyFromContext,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
}

func NewProxy(addr string, t string) Proxy {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		host = "127.0.0.1"
		port = "0"
	}
	iport, err := strconv.Atoi(port)
	if err != nil {
		iport = 0
	}
	p, ok := protoMap[t]
	if !ok {
		p = HTTP
	}
	return Proxy{
		IP:    net.ParseIP(host).To4(),
		Port:  uint16(iport),
		Proto: p,
	}
}

func HttpProxy(addr string) Proxy {
	return NewProxy(addr, "http")
}

func HttpsProxy(addr string) Proxy {
	return NewProxy(addr, "https")
}

func Socks4Proxy(addr string) Proxy {
	return NewProxy(addr, "socks4")
}

func Socks5Proxy(addr string) Proxy {
	return NewProxy(addr, "socks5")
}
