package pool

import (
	"context"
	"encoding/gob"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/history"
	"github.com/nfx/slrp/internal/qa"
	"github.com/nfx/slrp/ipinfo"
	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/ql/eval"
	"github.com/stretchr/testify/assert"
)

func TestSimpleAddAndRemove(t *testing.T) {
	pool, runtime := app.MockStartSpin(NewPool(history.NewHistory(), ipinfo.NoopIpInfo{
		Country: "Zimbabwe",
	}, &net.Dialer{}))
	defer runtime.Stop()

	ctx := context.Background()

	pool.Add(ctx, pmux.HttpProxy("127.0.0.1:8080"), 1*time.Second)
	assert.Equal(t, 1, pool.Len())

	pool.Remove(pmux.HttpProxy("127.0.0.1:8080"))
	assert.Equal(t, 0, pool.Len())
}

func TestMarshallAndUnmarshall(t *testing.T) {
	history := history.NewHistory()
	pool, first := app.MockStartSpin(NewPool(history, ipinfo.NoopIpInfo{
		Country: "Zimbabwe",
	}, &net.Dialer{}))
	defer first.Stop()

	ctx := context.Background()

	pool.Add(ctx, pmux.HttpProxy("127.0.0.1:8080"), 1*time.Second)
	assert.Equal(t, 1, pool.Len())

	raw, err := pool.MarshalBinary()
	assert.NoError(t, err)

	loaded := NewPool(history, ipinfo.NoopIpInfo{
		Country: "Zimbabwe",
	}, &net.Dialer{})
	err = loaded.UnmarshalBinary(raw)
	assert.NoError(t, err)

	_, second := app.MockStartSpin(loaded)
	defer second.Stop()

	// snapshots rely on runtime channels to compute
	assert.Equal(t, loaded.snapshot(), pool.snapshot())
}

type staticResponseClient struct {
	http.Response
	err error
}

func (r staticResponseClient) Do(req *http.Request) (*http.Response, error) {
	return &r.Response, r.err
}

func TestRoundTrip(t *testing.T) {
	pool, runtime := app.MockStartSpin(NewPool(history.NewHistory(), ipinfo.NoopIpInfo{
		Country: "Zimbabwe",
	}, &net.Dialer{}))
	defer runtime.Stop()

	pool.client = staticResponseClient{
		Response: http.Response{
			StatusCode: 200,
			Header:     http.Header{},
		},
	}

	proxy := pmux.Socks4Proxy("127.0.0.1:1")
	ctx := context.Background()

	pool.Add(ctx, proxy, 1*time.Second)
	assert.Equal(t, 1, pool.Len())

	// TODO: spin up test servers not to get to internet for no reason
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://anything", nil)
	res, err := pool.RoundTrip(req)
	assert.NoError(t, err)

	assert.Equal(t, 200, res.StatusCode)
}

func TestSession(t *testing.T) {
	proxy := pmux.Socks4Proxy("127.0.0.1:1")

	hist := history.NewHistory()
	pool, runtime := app.MockStartSpin(NewPool(hist, ipinfo.NoopIpInfo{
		Country: "Zimbabwe",
	}, &net.Dialer{}), hist)
	defer runtime.Stop()

	ctx := context.Background()
	pool.Add(ctx, proxy, 1*time.Second)
	assert.Equal(t, 1, pool.Len())

	pool.client = staticResponseClient{
		Response: http.Response{
			StatusCode: 200,
			Header:     http.Header{},
		},
	}

	err := pool.Session(ctx, func(ctx context.Context, c httpClient) error {
		req, _ := http.NewRequestWithContext(ctx, "GET", "http://something", nil)
		res, err := c.Do(req)
		if err != nil {
			return err
		}
		assert.Equal(t, 200, res.StatusCode)
		return nil
	})
	assert.NoError(t, err)
}

func TestHttpGet(t *testing.T) {
	pool, runtime := app.MockStartSpin(NewPool(history.NewHistory(), ipinfo.NoopIpInfo{
		Country: "Zimbabwe",
	}, &net.Dialer{}))
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
	stats := res.(*eval.QueryResult[ApiEntry])
	assert.Equal(t, 1, stats.Total)
	assert.Equal(t, "Zimbabwe", stats.Records[0].Country)
}

func load(t *testing.T) *Pool {
	qa.RunOnlyInDebug(t)
	f, err := os.Open("/tmp/harvester/pool")
	if err != nil {
		t.Fatal(err)
	}
	dec := gob.NewDecoder(f)
	pool := NewPool(history.NewHistory(), ipinfo.NoopIpInfo{
		Country: "Zimbabwe",
	}, &net.Dialer{})
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

func TestReceiveHalt(t *testing.T) {
	pool, runtime := app.MockStartSpin(NewPool(history.NewHistory(), ipinfo.NoopIpInfo{
		Country: "Zimbabwe",
	}, &net.Dialer{}))
	defer runtime.Stop()

	for i := 0; i < 33; i++ {
		pool.pressure <- i
	}

	v := <-pool.halt
	assert.Equal(t, time.Minute, v)
}

func TestCounterOnHalt(t *testing.T) {
	pool, runtime := app.MockStartSpin(NewPool(history.NewHistory(), ipinfo.NoopIpInfo{
		Country: "Zimbabwe",
	}, &net.Dialer{}))
	defer runtime.Stop()

	serial := <-pool.serial
	assert.Equal(t, 1, serial)

	now := time.Now()
	slowDown := time.Second * 1

	<-pool.serial
	pool.halt <- slowDown // <= bug
	serial = <-pool.serial

	assert.Equal(t, 3, serial)
	assert.GreaterOrEqual(t, time.Since(now), slowDown)

	serial = <-pool.serial

	assert.Equal(t, 4, serial)
}

func TestRandomFast(t *testing.T) {
	pool, runtime := app.MockStartSpin(NewPool(history.NewHistory(), ipinfo.NoopIpInfo{
		Country: "Zimbabwe",
	}, &net.Dialer{}))
	defer runtime.Stop()

	x := pmux.HttpProxy("127.0.0.1:1024")
	y := pmux.HttpProxy("127.0.0.1:1025")

	ctx := context.Background()
	pool.Add(ctx, x, time.Minute)
	pool.Add(ctx, y, time.Second)

	ctx2 := pool.RandomFast(ctx)
	found := pmux.GetProxyFromContext(ctx2)
	assert.Equal(t, y, found)
}

func TestRoundTripCtxErr(t *testing.T) {
	pool, runtime := app.MockStartSpin(NewPool(history.NewHistory(), ipinfo.NoopIpInfo{
		Country: "Zimbabwe",
	}, &net.Dialer{}))
	defer runtime.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	pool.Add(ctx, pmux.HttpProxy("127.0.0.1:1024"), time.Second)

	cancel()
	res, err := pool.RoundTrip((&http.Request{}).WithContext(ctx))
	assert.Nil(t, res)

	assert.EqualError(t, err, "context canceled")
}

func TestRoundTripNilResponseFromOut(t *testing.T) {
	pool, runtime := app.MockStartSpin(NewPool(history.NewHistory(), ipinfo.NoopIpInfo{
		Country: "Zimbabwe",
	}, &net.Dialer{}))
	defer runtime.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// instrument all shards for simplicity
	requests := make(chan request)
	for i := range pool.shards {
		pool.shards[i].requests = requests
	}

	done := make(chan int)
	go func() {
		res, err := pool.RoundTrip((&http.Request{
			Header: http.Header{},
		}).WithContext(ctx))
		assert.NotNil(t, res)
		assert.NoError(t, err)
		assert.Equal(t, 200, res.StatusCode)
		t.Log("done")
		<-done
	}()

	r1 := <-requests
	assert.Equal(t, 1, r1.serial)
	assert.Equal(t, 1, r1.attempt)
	r1.out <- nil

	r2 := <-requests
	assert.Equal(t, 1, r2.serial)
	assert.Equal(t, 2, r2.attempt)
	r2.out <- &http.Response{StatusCode: 200}
	done <- 200
}
