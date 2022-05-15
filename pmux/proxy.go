package pmux

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

type proto uint16

const (
	// don't change the order!!!
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

// uint64 = uint32 + uint16 + uint16 (padding for alignment)
type Proxy uint64

func (p Proxy) IP() net.IP {
	ip := p >> 32
	a := ip >> 24 & 0xff
	b := ip >> 16 & 0xff
	c := ip >> 8 & 0xff
	d := ip & 0xff
	return net.IPv4(byte(a), byte(b), byte(c), byte(d))
}

func (p Proxy) Address() string {
	ip := p >> 32
	a := ip >> 24 & 0xff
	b := ip >> 16 & 0xff
	c := ip >> 8 & 0xff
	d := ip & 0xff
	return fmt.Sprintf("%d.%d.%d.%d:%d",
		a, b, c, d, p.Port())
}

func (p Proxy) Port() uint16 {
	return uint16(p >> 16 & 0xffff)
}

func (p Proxy) Proto() proto {
	return proto(p & 0xffff)
}

func (p Proxy) Scheme() string {
	return reverseProtoMap[p.Proto()]
}

func (p Proxy) URL() *url.URL {
	return &url.URL{
		Host:   p.Address(),
		Scheme: p.Scheme(),
	}
}

func (p Proxy) String() string {
	return fmt.Sprintf("%s://%s", p.Scheme(), p.Address())
}

func (p Proxy) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, p.String())), nil
}

type ckey int

const ProxyURL ckey = iota

func (p Proxy) InContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ProxyURL, p)
}

func (p Proxy) Valid() bool {
	return p.Port() != 0
}

func (p Proxy) IsTunnel() bool {
	return p.Proto() == SOCKS4 || p.Proto() == SOCKS5
}

func (p Proxy) Bucket(buckets int) int {
	bucket := int(p) % buckets
	if bucket < 0 {
		return bucket * -1
	}
	return bucket
}

func GetProxyFromContext(ctx context.Context) Proxy {
	p := ctx.Value(ProxyURL)
	if p == nil {
		return 0
	}
	proxy, ok := p.(Proxy)
	if !ok {
		return 0
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
	if p == 0 || !p.IsTunnel() {
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
	if p == 0 {
		return nil, nil
	}
	if p.IsTunnel() {
		// handled in DialContext
		return nil, nil
	}
	//return p.URL(), nil
	return &url.URL{
		Host:   p.Address(),
		Scheme: "http",
	}, nil
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
	addrPort, err := netip.ParseAddrPort(addr)
	if err != nil {
		return 0
	}
	p, ok := protoMap[t]
	if !ok {
		p = HTTP
	}
	var ipv4u64 uint64
	ipv4bytes := addrPort.Addr().As4()
	for i := 0; i < 4; i++ {
		ipv4u64 |= uint64(ipv4bytes[i])
		if i < 3 {
			ipv4u64 <<= 8
		}
	}
	ip := ipv4u64 << 32
	port := uint64(addrPort.Port()) << 16
	return Proxy(ip + port + uint64(p))
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
