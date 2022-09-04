package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDocumentWriteInJsVm(t *testing.T) {
	ctx := context.Background()
	wrote, err := documentWriteInJsVm(ctx, `document.write(atob("MTQ="))`)
	assert.NoError(t, err)
	assert.Equal(t, "14", wrote)
}

func TestDocumentWriteInJsVm_Error(t *testing.T) {
	ctx := context.Background()
	_, err := documentWriteInJsVm(ctx, `atob("!@#$%")`)
	assert.EqualError(t, err, "GoError: illegal base64 data at input byte 0 "+
		"at github.com/nfx/slrp/sources.documentWriteInJsVm.func1 (native)")
}

func TestProxyNova_Failures(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	server := httptest.NewServer(http.FileServer(http.Dir("./testdata/proxynova")))
	defer server.Close()
	proxyNovaPrefix = server.URL
	countries = []string{"fail"}

	feed := ByID(7).Feed(ctx, http.DefaultClient)
	found := consumeSource(ctx, feed)
	assert.Equal(t, 4, len(found))
	err := feed.Err()
	// matching on error string is a bit hard because of random port
	assert.Error(t, err)
}

func TestProxyNova(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir("./testdata/proxynova")))
	defer server.Close()
	proxyNovaPrefix = server.URL
	countries = []string{"mock"}
	testSource(t, func(ctx context.Context) Src {
		return ByID(7).Feed(ctx, http.DefaultClient)
	}, 6)
}
