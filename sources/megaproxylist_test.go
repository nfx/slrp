package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMegaproxylist(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir(
		fmt.Sprintf("./testdata/megaproxylist/megaproxylist-csv-%s_SDACH.zip", time.Now().Format("20060102")),
	)))
	defer server.Close()
	megaproxylistUrl = fmt.Sprintf("%s/page", server.URL)
	fmt.Println(megaproxylistUrl)
	testSource(t, func(ctx context.Context) Src {
		return ByID(69).Feed(ctx, http.DefaultClient)
	}, 3) // 3 is fake
}
