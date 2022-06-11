package pool

import (
	"context"
	"encoding/gob"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/history"
	"github.com/nfx/slrp/internal/qa"
	"github.com/nfx/slrp/pmux"
	"github.com/stretchr/testify/assert"
)

func TestSimpleAddAndRemove(t *testing.T) {
	pool, runtime := app.MockStartSpin(NewPool(history.NewHistory()))
	defer runtime.Stop()

	ctx := context.Background()

	pool.Add(ctx, pmux.HttpProxy("127.0.0.1:8080"), 1*time.Second)
	assert.Equal(t, 1, pool.Len())

	pool.Remove(pmux.HttpProxy("127.0.0.1:8080"))
	assert.Equal(t, 0, pool.Len())
}

func TestMarshallAndUnmarshall(t *testing.T) {
	history := history.NewHistory()
	pool, first := app.MockStartSpin(NewPool(history))
	defer first.Stop()

	ctx := context.Background()

	pool.Add(ctx, pmux.HttpProxy("127.0.0.1:8080"), 1*time.Second)
	assert.Equal(t, 1, pool.Len())

	raw, err := pool.MarshalBinary()
	assert.NoError(t, err)

	loaded := NewPool(history)
	err = loaded.UnmarshalBinary(raw)
	assert.NoError(t, err)

	_, second := app.MockStartSpin(loaded)
	defer second.Stop()

	// snapshots rely on runtime channels to compute
	assert.Equal(t, loaded.snapshot(), pool.snapshot())
}

func TestRoundTrip(t *testing.T) {
	pool, runtime := app.MockStartSpin(NewPool(history.NewHistory()))
	defer runtime.Stop()

	var proxy pmux.Proxy
	defer pmux.SetupHttpProxy(&proxy)()
	ctx := context.Background()

	pool.Add(ctx, proxy, 1*time.Second)
	assert.Equal(t, 1, pool.Len())

	// TODO: spin up test servers not to get to internet for no reason
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://httpbin.org/get", nil)
	res, err := pool.RoundTrip(req)
	assert.NoError(t, err)

	assert.Equal(t, 200, res.StatusCode)
}

func TestSessionHistory(t *testing.T) {
	var proxy pmux.Proxy
	defer pmux.SetupHttpProxy(&proxy)()

	hist := history.NewHistory()
	pool, runtime := app.MockStartSpin(NewPool(hist), hist)
	defer runtime.Stop()

	ctx := context.Background()
	pool.Add(ctx, proxy, 1*time.Second)
	assert.Equal(t, 1, pool.Len())

	err := pool.Session(ctx, func(ctx context.Context, c *http.Client) error {
		req, _ := http.NewRequestWithContext(ctx, "GET", "http://httpbin.org/get", nil)
		res, err := c.Do(req)
		if err != nil {
			return err
		}
		assert.Equal(t, 200, res.StatusCode)
		return nil
	})
	assert.NoError(t, err)

	res, err := hist.HttpGetByID("1", nil)
	assert.NoError(t, err)
	req := res.(history.Request)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "http://httpbin.org/get", req.URL)
}

func TestHttpGet(t *testing.T) {
	pool, runtime := app.MockStartSpin(NewPool(history.NewHistory()))
	defer runtime.Stop()

	ctx := context.Background()

	pool.Add(ctx, pmux.HttpProxy("127.0.0.1:8080"), 1*time.Second)
	assert.Equal(t, 1, pool.Len())

	res, err := pool.HttpGet(&http.Request{
		URL: &url.URL{
			RawQuery: "filter=Offered:1",
		},
	})
	assert.NoError(t, err)
	stats := res.(PoolStats)
	assert.Equal(t, 1, stats.Total)
}

func load(t *testing.T) *Pool {
	qa.RunOnlyInDebug(t)
	f, err := os.Open("/tmp/harvester/pool")
	if err != nil {
		t.Fatal(err)
	}
	dec := gob.NewDecoder(f)
	pool := NewPool(history.NewHistory())
	dec.Decode(pool)
	return pool
}

func TestSelection(t *testing.T) {
	mctx := app.MockCtx()
	defer mctx.Cancel()
	pool := load(t)
	pool.Start(mctx)

	ctx := mctx.Ctx()
	log := app.Log.From(ctx)

	seen := map[string]int{}
	shard := pool.shards[0]

	dump := func() {
		defaultSorting(shard.Entries)
		all := []string{}
		for _, v := range shard.Entries[0:10] {
			all = append(all, v.String())
		}
		log.Info().Msgf("all:\n%s", strings.Join(all, "\n"))
	}
	for i := 0; i < len(shard.Entries); i++ {
		dump()
		e := shard.firstAvailableProxy(request{
			serial: i,
			in:     (&http.Request{}).WithContext(ctx),
		})
		seen[e.Proxy.String()] = seen[e.Proxy.String()] + 1
		var err error
		if seen[e.Proxy.String()] > 1 {
			err = fmt.Errorf("seen %d times", seen[e.Proxy.String()])
		}
		log.Info().
			Err(err).
			Msg(e.String())
		e.MarkSuccess()
	}
	// all = []string{}
	// for _, v := range shard.Entries {
	// 	all = append(all, v.String())
	// }
	// log.Info().Msgf("all:\n%s", strings.Join(all, "\n"))
	t.Fail()
}
