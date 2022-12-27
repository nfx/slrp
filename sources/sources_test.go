package sources

import (
	"context"
	"net/http"
	"strings"
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

func TestNoDuplicateIDs(t *testing.T) {
	seen := map[int][]string{}
	for _, v := range Sources {
		seen[v.ID] = append(seen[v.ID], v.Name())
	}
	for _, v := range Sources {
		if len(seen[v.ID]) > 1 {
			t.Fatalf("ID %d is occupied by %s", v.ID,
				strings.Join(seen[v.ID], " and "))
		}
	}
}

func TestJetkai(t *testing.T) {
	src := ByName("jetkai")
	assert.Equal(t, "jetkai", src.Name())
	testSource(t, func(ctx context.Context) Src {
		return src.Feed(ctx, http.DefaultClient)
	}, 1000)
}
