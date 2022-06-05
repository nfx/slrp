package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPremproxyFixtures(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir("./testdata/premproxy")))
	defer server.Close()
	premProxyHttpPages = []string{fmt.Sprintf("%s/http", server.URL)}
	premProxySocksPages = []string{fmt.Sprintf("%s/socks", server.URL)}
	testSource(t, func(ctx context.Context) Src {
		return ByID(5).Feed(ctx, http.DefaultClient)
	}, 4)
}
