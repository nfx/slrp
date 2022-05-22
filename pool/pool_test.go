package pool

import (
	"context"
	"encoding/gob"
	"fmt"
	"net/http"
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
	pool := NewPool(history.NewHistory())
	runtime := app.Singletons{"pool": pool}.MockStart()
	defer runtime.Stop()
	runtime["pool"].Spin()

	ctx := context.Background()

	pool.Add(ctx, pmux.HttpProxy("127.0.0.1:8080"), 1*time.Second)
	assert.Equal(t, 1, pool.Len())

	pool.Remove(pmux.HttpProxy("127.0.0.1:8080"))
	assert.Equal(t, 0, pool.Len())
}

func TestMarshallAndUnmarshall(t *testing.T) {
	pool := NewPool(history.NewHistory())
	firstRuntime := app.Singletons{"pool": pool}.MockStart()
	defer firstRuntime.Stop()
	firstRuntime["pool"].Spin()

	ctx := context.Background()

	pool.Add(ctx, pmux.HttpProxy("127.0.0.1:8080"), 1*time.Second)
	assert.Equal(t, 1, pool.Len())

	raw, err := pool.MarshalBinary()
	assert.NoError(t, err)

	loaded := NewPool(pool.history)
	err = loaded.UnmarshalBinary(raw)
	assert.NoError(t, err)

	secondRuntime := app.Singletons{"pool": loaded}.MockStart()
	defer secondRuntime.Stop()

	// snapshots rely on runtime channels to compute
	assert.Equal(t, loaded.snapshot(), pool.snapshot())
}

func TestRoundTrip(t *testing.T) {
	t.Skip("TODO: FIXME!")
	pool := NewPool(history.NewHistory())
	runtime := app.Singletons{"pool": pool}.MockStart()
	defer runtime.Stop()
	runtime["pool"].Spin()

	var proxy pmux.Proxy
	defer pmux.SetupProxy(&proxy)()
	ctx := context.Background()

	pool.Add(ctx, proxy, 1*time.Second)
	assert.Equal(t, 1, pool.Len())

	// TODO: spin up test servers not to get to internet for no reason
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://httpbin.org/get", nil)
	res, err := pool.RoundTrip(req)
	assert.NoError(t, err)

	assert.Equal(t, 200, res.StatusCode)
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
