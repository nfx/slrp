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

func Test_retriableGenerator_Len(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	g := gen(func() ([]pmux.Proxy, error) {
		return []pmux.Proxy{
			pmux.HttpProxy("127.0.0.1:23"),
			pmux.HttpProxy("127.0.0.1:24"),
		}, nil
	})

	<-g.Generate(ctx)
	assert.Equal(t, 2, g.Len())
}

func Test_mergeSrc_Len(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	canComputeA := make(chan bool)
	canComputeB := make(chan bool)
	m := merged().refresh(func() ([]pmux.Proxy, error) {
		<-canComputeA
		return []pmux.Proxy{
			pmux.HttpProxy("127.0.0.1:1024"),
			pmux.HttpProxy("127.0.0.1:1025"),
		}, nil
	}).refresh(func() ([]pmux.Proxy, error) {
		<-canComputeB
		return []pmux.Proxy{
			pmux.HttpProxy("127.0.0.1:2048"),
			pmux.HttpProxy("127.0.0.1:2049"),
			pmux.HttpProxy("127.0.0.1:2050"),
		}, nil
	})

	// in the beginning, we know of two sources
	assert.Equal(t, 2, m.Len())

	// consume in background and notify test can assert
	canAssertA := make(chan bool)
	canAssertB := make(chan bool)
	go func() {
		for v := range m.Generate(ctx) {
			switch v.Proxy { // switch on the first items in results
			case pmux.HttpProxy("127.0.0.1:1024"):
				canAssertA <- true
			case pmux.HttpProxy("127.0.0.1:2048"):
				canAssertB <- true
			}
			t.Logf("received: %v", v)
		}
	}()

	canComputeA <- true // can compute
	<-canAssertA        // can assert
	assert.Equal(t, 3, m.Len())

	canComputeB <- true // can compute
	<-canAssertB        // can assert
	assert.Equal(t, 5, m.Len())
}

func Test_mergeSrc_StopsOnDone(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)

	m := merged().refresh(func() ([]pmux.Proxy, error) {
		return []pmux.Proxy{
			pmux.HttpProxy("127.0.0.1:1024"),
			pmux.HttpProxy("127.0.0.1:1025"),
			pmux.HttpProxy("127.0.0.1:1026"),
		}, nil
	}).refresh(func() ([]pmux.Proxy, error) {
		return []pmux.Proxy{
			pmux.HttpProxy("127.0.0.1:2048"),
			pmux.HttpProxy("127.0.0.1:2049"),
		}, nil
	})

	ch := m.Generate(ctx)

	v, ok := <-ch
	assert.True(t, ok, "channel must not be closed")
	assert.NotNil(t, v)

	cancel() // stop the world

	_, ok = <-ch
	t.Logf("channel closed? %v", ok)

	_, ok = <-ch
	assert.False(t, ok, "channel must be closed")
}
