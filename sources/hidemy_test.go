package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHidemy(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir("./testdata/hidemy")))
	defer server.Close()
	hidemyNamePages = []string{fmt.Sprintf("%s/page", server.URL)}
	testSource(t, func(ctx context.Context) Src {
		return ByID(24).Feed(ctx, http.DefaultClient)
	}, 5)
}
