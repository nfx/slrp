package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxydbFixtures(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir("./testdata/proxydb")))
	defer server.Close()
	proxyDbPages = map[string]string{
		fmt.Sprintf("%s/page", server.URL): "http",
	}
	testSource(t, func(ctx context.Context) Src {
		return ByID(13).Feed(ctx, http.DefaultClient)
	}, 6)
}
