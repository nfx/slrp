package pmux

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/rs/zerolog/log"
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
	ctx = context.WithValue(ctx, ProxyURL, "lalala")
	proxy := GetProxyFromContext(ctx)
	assert.Equal(t, Proxy(0), proxy)
}

func TestContextualHttpTransport(t *testing.T) {
	transport := ContextualHttpTransport()
	assert.NotNil(t, transport)
}

func TestVerifyProxyInContext(t *testing.T) {
	var proxy Proxy
	defer SetupHttpProxy(&proxy)()
	c := &http.Client{
		Transport: &http.Transport{
			DialContext: dialProxiedConnection,
			Proxy:       pickHttpProxyFromContext,
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = proxy.InContext(ctx)
	get := func(url string) *http.Response {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			log.Printf("[ERROR] %s", err)
			return nil
		}
		res, err := c.Do(req)
		if err != nil {
			log.Printf("[ERROR] %s", err)
			return nil
		}
		return res
	}
	log.Printf("HTTP: %#v", get("http://google.com"))
	log.Printf("HTTP: %#v", get("https://ifconfig.me"))
}
