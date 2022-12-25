package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxylistDownloadFixtures(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir("./testdata/proxyListDownload")))
	defer server.Close()
	proxyListPages = []string{fmt.Sprintf("%s/page", server.URL)}
	testSource(t, func(ctx context.Context) Src {
		return ByID(55).Feed(ctx, http.DefaultClient)
	}, 3)
}
