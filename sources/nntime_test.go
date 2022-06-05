package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNNtimeFixtures(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir("./testdata/nntime")))
	defer server.Close()
	nntimePages = 1
	nntimePattern = fmt.Sprintf("%s/page-%%02d", server.URL)
	testSource(t, func(ctx context.Context) Src {
		return ByID(4).Feed(ctx, http.DefaultClient)
	}, 3)
}
