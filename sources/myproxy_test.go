package sources

import (
	"context"
	"net/http"
	"testing"
)

func TestMyproxy(t *testing.T) {
	testSource(t, func(ctx context.Context) Src {
		return ByID(3).Feed(ctx, http.DefaultClient)
	}, 900)
}
