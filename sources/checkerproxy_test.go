package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckerproxy(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir("./testdata/checkerproxy")))
	defer server.Close()
	checkerProxyURL = fmt.Sprintf("%s/page", server.URL)
	testSource(t, func(ctx context.Context) Src {
		return ByID(1).Feed(ctx, http.DefaultClient)
	}, 3)
}
