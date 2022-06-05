package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFreeproxyczFixtures(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir("./testdata/freeproxycz")))
	defer server.Close()
	freeProxyCzPages = []string{fmt.Sprintf("%s/page", server.URL)}
	testSource(t, func(ctx context.Context) Src {
		return ByID(2).Feed(ctx, http.DefaultClient)
	}, 4)
}
