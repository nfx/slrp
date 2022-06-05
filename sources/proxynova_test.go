package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxyNova(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir("./testdata/proxynova")))
	defer server.Close()
	proxyNovaPrefix = server.URL
	countries = []string{"mock"}
	testSource(t, func(ctx context.Context) Src {
		return ByID(7).Feed(ctx, http.DefaultClient)
	}, 6)
}
