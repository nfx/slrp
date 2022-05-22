package sources

import (
	"context"
	"net/http"
	"testing"
)

func TestProxyNova(t *testing.T) {
	t.Skip("too long")
	testSource(t, func(ctx context.Context) Src {
		return ByID(7).Feed(ctx, http.DefaultClient)
	}, 1000)
}
