package pmux

import (
	"context"
	"net/http"
	"testing"

	"github.com/rs/zerolog/log"
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
