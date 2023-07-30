package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nfx/slrp/internal/qa"
	"github.com/rs/zerolog/log"
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

func TestPremproxy(t *testing.T) {
	qa.RunOnlyInDebug(t)
	ctx := context.Background()
	src := premproxy(ctx, &http.Client{})
	seen := map[string]int{}
	for x := range src.Generate(ctx) {
		y := x.Proxy.String()
		seen[y] = seen[y] + 1
	}
	log.Printf("found: %d", len(seen))
}
