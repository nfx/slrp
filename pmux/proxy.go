package pmux

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/proxy"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.DurationFieldUnit = time.Second
}

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

func (p Proxy) AsHttp() Proxy {
	ip := uint64(p>>32) << 32
	port := uint64(p.Port()) << 16
	return Proxy(ip + port + uint64(HTTP))
}

func (p Proxy) AsHttps() Proxy {
	ip := uint64(p>>32) << 32
	port := uint64(p.Port()) << 16
	return Proxy(ip + port + uint64(HTTPS))
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

const proxyURL ckey = iota

func (p Proxy) InContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, proxyURL, p)
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

// MustNewGetRequest is a utility method for testing
func (p Proxy) MustNewGetRequest(url string) *http.Request {
	ctx := p.InContext(context.Background())
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		panic(err)
	}
	return req
}

func GetProxyFromContext(ctx context.Context) Proxy {
	p := ctx.Value(proxyURL)
	if p == nil {
		return 0
	}
	proxy, ok := p.(Proxy)
	if !ok {
		return 0
	}
	return proxy
}

var DefaultHttpClient = &http.Client{
	// TODO: convert other usages ContextualHttpTransport to this
	// and harmonise parameters
	Transport: ContextualHttpTransport(),
	Timeout:   10 * time.Second,
}

var DefaultTlsConfig = &tls.Config{
	InsecureSkipVerify: true,
	NextProtos:         []string{"http/1.1"},
}

var DefaultDialer = &net.Dialer{
	// TODO: a) configure this timeout globally
	// TODO: b) configure this per-proxy (we know their speed)
	Timeout:   5 * time.Second,
	KeepAlive: 0,
}

// experiment with net.Dialer.Control to bypass TCP fingerprinting
// http://witch.valdikss.org.ru/
// https://en.wikipedia.org/wiki/TCP/IP_stack_fingerprinting
// https://stackoverflow.com/a/52426887/277035
func dialProxiedConnection(ctx context.Context, network, addr string) (net.Conn, error) {
	p := GetProxyFromContext(ctx)
	switch p.Proto() {
	case SOCKS4, SOCKS5:
		dialer, err := proxy.FromURL(p.URL(), proxy.Direct)
		if err != nil {
			return nil, err
		}
		conn, err := dialer.Dial(network, addr)
		if err != nil {
			return nil, fmt.Errorf("dial socks: %w", err)
		}
		if strings.HasSuffix(addr, ":80") {
			// this is HTTP connection, no need to TLS it
			// TODO: figure out a better way of determining this
			return conn, nil
		}
		return tls.Client(conn, DefaultTlsConfig), nil
	case HTTPS:
		conn, err := DefaultDialer.DialContext(ctx, network, addr)
		if err != nil {
			return nil, fmt.Errorf("dial https: %w", err)
		}
		return tls.Client(conn, DefaultTlsConfig), nil
	default:
		// use normal connection establishment in one of two cases:
		// a) no proxy is specified
		// b) HTTP proxy (handled higher on the stack)
		return DefaultDialer.DialContext(ctx, network, addr)
	}
}

func ProxyFromContext(r *http.Request) (*url.URL, error) {
	p := GetProxyFromContext(r.Context())
	if p == 0 {
		return nil, nil
	}
	// if p.IsTunnel() {
	// 	// handled in DialContext
	// 	return nil, nil
	// }
	// TODO: free-proxy.cz is not liking HTTPS dialer, so it needs only HTTP forwarder
	return p.URL(), nil
}

var contextualTransport = &http.Transport{
	// If DialTLSContext is set, the Dial and DialContext hooks are not used for HTTPS
	// requests and the TLSClientConfig and TLSHandshakeTimeout
	// are ignored. The returned net.Conn is assumed to already be
	// past the TLS handshake.
	// DialTLSContext:      dialProxiedConnection,
	TLSClientConfig: DefaultTlsConfig,
	// TLSHandshakeTimeout: DefaultDialer.Timeout,
	Proxy: ProxyFromContext,
	// DisableKeepAlives:   true,
	// MaxIdleConns:        0,
}

func ContextualHttpTransport() *http.Transport {
	return contextualTransport
}

func NewProxyFromURL(url string) Proxy {
	split := strings.Split(url, "://")
	if len(split) != 2 {
		return 0
	}
	return NewProxy(split[1], split[0])
}

func NewProxy(addr string, t string) Proxy {
	// Check if the address is valid or contains "[::]"; This happens when running inside a docker container
	// It means that the address is listening on all interfaces but via IPv6, which is not supported by the
	// proxy package(or so). Hence we replace it with 0.0.0.0
	if strings.Contains(addr, "[::]") {
		// Set it to 0.0.0.0 but maintain the port
		fmt.Println("Encountered [::]: in address, replacing with 0.0.0.0")
		addr = strings.Replace(addr, "[::]", "0.0.0.0", 1)
	}

	addrPort, err := netip.ParseAddrPort(addr)
	if err != nil {
		fmt.Println("Error parsing address:", err)
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
