package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMegaproxylist(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir(
		fmt.Sprintf("./testdata/megaproxylist/test.zip"),
	)))
	defer server.Close()
	megaproxylistUrl = fmt.Sprintf("%s/page", server.URL)
	fmt.Println(megaproxylistUrl)
	testSource(t, func(ctx context.Context) Src {
		return ByID(69).Feed(ctx, http.DefaultClient)
	}, 3) // 3 is fake
}
