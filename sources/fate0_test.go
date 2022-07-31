package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFate0(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir("./testdata/fate0")))
	defer server.Close()
	fateZeroURL = fmt.Sprintf("%s/page", server.URL)
	testSource(t, func(ctx context.Context) Src {
		return ByID(22).Feed(ctx, http.DefaultClient)
	}, 5)
}
