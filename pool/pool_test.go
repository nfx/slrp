package pool

import (
	"encoding/gob"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/history"
)

func load(t *testing.T) *Pool {
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
