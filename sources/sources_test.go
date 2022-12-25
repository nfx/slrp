package sources

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/nfx/slrp/pmux"
	"github.com/stretchr/testify/assert"
)

func consumeSource(ctx context.Context, feed Src) (found []pmux.Proxy) {
	for proxy := range feed.Generate(ctx) {
		found = append(found, proxy)
	}
	return
}

func testSource(t *testing.T, cb func(context.Context) Src, atLeast int) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	feed := cb(ctx)
	found := consumeSource(ctx, feed)
	assert.GreaterOrEqual(t, len(found), atLeast)
	err := feed.Err()
	assert.NoError(t, err)
}

func TestJetkai(t *testing.T) {
	src := ByName("jetkai")
	assert.Equal(t, "jetkai", src.Name())
	testSource(t, func(ctx context.Context) Src {
		return src.Feed(ctx, http.DefaultClient)
	}, 2500)
}
