package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebanet(t *testing.T) {
	t.Skip() // last updated only on 22.01.2023
	server := httptest.NewServer(http.FileServer(http.Dir("./testdata/webanet")))
	defer server.Close()
	webanetURL = fmt.Sprintf("%s/page", server.URL)
	testSource(t, func(ctx context.Context) Src {
		return ByID(16).Feed(ctx, http.DefaultClient)
	}, 4)
}
