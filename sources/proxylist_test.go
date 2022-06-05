package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxylistFixtures(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir("./testdata/proxylist")))
	defer server.Close()
	proxyListPages = []string{fmt.Sprintf("%s/page", server.URL)}
	testSource(t, func(ctx context.Context) Src {
		return ByID(6).Feed(ctx, http.DefaultClient)
	}, 8)
}
