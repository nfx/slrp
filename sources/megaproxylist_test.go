package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMegaproxylist(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir("./testdata/megaproxylist")))
	defer server.Close()
	megaproxylistUrl = fmt.Sprintf("%s/test.zip", server.URL)
	testSource(t, func(ctx context.Context) Src {
		return ByID(65).Feed(ctx, http.DefaultClient)
	}, 3)
}
