package sources

import (
	"context"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/nfx/slrp/internal/qa"
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

func TestThespeedx(t *testing.T) {
	src := ByName("speedx")
	assert.Equal(t, "speedx", src.Name())
	testSource(t, func(ctx context.Context) Src {
		return src.Feed(ctx, http.DefaultClient)
	}, 4000)
}

func TestJetkai(t *testing.T) {
	src := ByName("jetkai")
	assert.Equal(t, "jetkai", src.Name())
	testSource(t, func(ctx context.Context) Src {
		return src.Feed(ctx, http.DefaultClient)
	}, 3000)
}

func TestSslFreeProxies(t *testing.T) {
	src := ByID(18)
	assert.Equal(t, "sslproxies.org", src.Name())
	testSource(t, func(ctx context.Context) Src {
		return src.Feed(ctx, http.DefaultClient)
	}, 100)
}

func TestProxyLists(t *testing.T) {
	src := ByID(11)
	assert.Equal(t, "proxylists.net", src.Name())
	testSource(t, func(ctx context.Context) Src {
		return src.Feed(ctx, http.DefaultClient)
	}, 100)
}

func TestPremproxy(t *testing.T) {
	qa.RunOnlyInDebug(t)
	ctx := context.Background()
	src := premproxy(ctx, &http.Client{})
	seen := map[string]int{}
	for x := range src.Generate(ctx) {
		y := x.String()
		seen[y] = seen[y] + 1
	}
	log.Printf("found: %d", len(seen))
}
