package sources

import (
	"context"
	"net/http"
	"testing"
)

func TestWebanet(t *testing.T) {
	testSource(t, func(ctx context.Context) Src {
		return ByID(16).Feed(ctx, http.DefaultClient)
	}, 2000)
}
