package sources

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/nfx/slrp/pmux"
	"github.com/stretchr/testify/assert"
)

func TestSimpleGen(t *testing.T) {
	testSource(t, func(ctx context.Context) Src {
		feed := simpleGen(func(ctx context.Context, c *http.Client) ([]pmux.Proxy, error) {
			return []pmux.Proxy{
				pmux.HttpProxy("127.0.0.1:8081"),
			}, nil
		})
		return feed(ctx, http.DefaultClient)
	}, 1)
}

func TestRetriableSrcError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	src := gen(func() ([]pmux.Proxy, error) {
		return nil, errors.New("random error")
	})
	found := consumeSource(ctx, src)
	assert.Equal(t, 0, len(found))

	err := src.Err()
	assert.EqualError(t, err, "random error")
}

func TestSkippableSrcError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	src := gen(func() ([]pmux.Proxy, error) {
		return nil, skipErr(
			newErr("base msg to be skipped", intEC{"a", 1}),
			strEC{"b", "c"})
	})
	found := consumeSource(ctx, src)
	assert.Equal(t, 0, len(found))

	err := src.Err()
	assert.EqualError(t, err, "base msg to be skipped a=1 b=c (skip)")
}

func TestIntermediateSrcError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	src := gen(func() ([]pmux.Proxy, error) {
		return nil, newErr("base msg", intEC{"a", 1})
	})
	found := consumeSource(ctx, src)
	assert.Equal(t, 0, len(found))

	err := src.Err()
	assert.EqualError(t, err, "base msg a=1")
}
