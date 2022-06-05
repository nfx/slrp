package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSpysoneFixtures(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir("./testdata/spysone")))
	defer server.Close()
	spysOneSleep = func() {}
	spysOnePageURL = fmt.Sprintf("%s/page", server.URL)
	testSource(t, func(ctx context.Context) Src {
		return ByID(14).Feed(ctx, http.DefaultClient)
	}, 4)
}
