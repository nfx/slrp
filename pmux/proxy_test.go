package pmux

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProxy(t *testing.T) {
	p := NewProxy("1.2.4.5:8731", "http")
	if p.String() != "http://1.2.4.5:8731" {
		t.Errorf("http proxy string failed")
	}
	p = HttpsProxy("1.2.4.6:8931")
	if p.String() != "https://1.2.4.6:8931" {
		t.Errorf("https proxy string failed")
	}
}

func TestProxyIP(t *testing.T) {
	assert.Equal(t, net.IPv4(1, 2, 3, 4), HttpProxy("1.2.3.4:56789").IP())
}

func TestProxyValid(t *testing.T) {
	assert.Equal(t, true, HttpProxy("1.2.3.4:56789").Valid())
}

func TestProxyBucket(t *testing.T) {
	assert.Equal(t, 4, HttpProxy("127.0.0.1:23456").Bucket(10))
}

func TestProxyURL(t *testing.T) {
	assert.Equal(t, "socks5://1.2.3.4:56789", Socks5Proxy("1.2.3.4:56789").URL().String())
}

func TestProxyMarlshalJSON(t *testing.T) {
	x, err := Socks4Proxy("1.2.3.4:56789").MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `"socks4://1.2.3.4:56789"`, string(x))
}

func TestGetProxyFromContext(t *testing.T) {
	proxy := GetProxyFromContext(context.Background())
	assert.Equal(t, Proxy(0), proxy)
}

func TestGetProxyFromContextInvalidType(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, proxyURL, "lalala")
	proxy := GetProxyFromContext(ctx)
	assert.Equal(t, Proxy(0), proxy)
}

func TestContextualHttpTransport(t *testing.T) {
	transport := ContextualHttpTransport()
	assert.NotNil(t, transport)
}

func TestDialProxiedConnection(t *testing.T) {
	p := NewProxy("1.2.4.5:8731", "http")
	r := p.MustNewGetRequest("https://ifconfig.me")
	_, err := dialProxiedConnection(r.Context(), "tcp", "127.0.0.1:")
	assert.Error(t, err)
}

func TestDialProxiedConnection_HTTPS(t *testing.T) {
	p := NewProxy("1.2.4.5:8731", "https")
	r := p.MustNewGetRequest("https://ifconfig.me")
	_, err := dialProxiedConnection(r.Context(), "tcp", "127.0.0.1:")
	assert.Error(t, err)
}

func TestDialProxiedConnection_SOCKS(t *testing.T) {
	p := Socks5Proxy("127.0.0.1:0")
	r := p.MustNewGetRequest("https://ifconfig.me")
	_, err := dialProxiedConnection(r.Context(), "tcp", "127.0.0.1:")
	assert.Error(t, err)
}

func TestPickProxyFromContext(t *testing.T) {
	p := HttpProxy("127.0.0.1:0")
	r := p.MustNewGetRequest("https://ifconfig.me")
	u, _ := ProxyFromContext(r)
	assert.Equal(t, u.String(), p.String())
}

func TestPickProxyFromContext_Tunnel(t *testing.T) {
	p := Socks5Proxy("127.0.0.1:0")
	r := p.MustNewGetRequest("https://ifconfig.me")
	u, err := ProxyFromContext(r)
	assert.Nil(t, u)
	assert.NoError(t, err)
}

func TestPickProxyFromContext_NoProxy(t *testing.T) {
	u, err := ProxyFromContext(&http.Request{})
	assert.Nil(t, u)
	assert.NoError(t, err)
}
