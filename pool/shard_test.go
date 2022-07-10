package pool

import (
	"net/http"
	"testing"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/stretchr/testify/assert"
)

func TestShardHandleReanimate(t *testing.T) {
	now = func() time.Time {
		return ti(0, 2, 0)
	}
	s := &shard{
		Entries: []entry{
			{ReanimateAfter: ti(0, 1, 0)},
		},
	}
	s.init(make(chan work))
	s.minute = time.NewTicker(500*time.Millisecond)

	ctx := app.MockCtx()
	defer ctx.Cancel()

	go s.main(ctx)
	<-ctx.Wait

	assert.True(t, s.Entries[0].Ok)
}

func TestShardHandleReply(t *testing.T) {
	now = func() time.Time {
		return ti(0, 2, 0)
	}
	s := &shard{
		Entries: []entry{
			{ReanimateAfter: ti(0, 1, 0)},
		},
	}
	s.init(make(chan work))

	ctx := app.MockCtx()
	defer ctx.Cancel()

	go s.main(ctx)

	out := make(chan *http.Response)
	s.reply <- reply{
		r: request{
			in: &http.Request{},
			serial: 123,
			out: out,
			attempt: 10,
		},
		response: &http.Response{
			StatusCode: 418,
			Status: "I'm a teapot",
		},
		e: &entry{},
	}
	response := <-out

	assert.Equal(t, 429, response.StatusCode)
	assert.Equal(t, "123", response.Header.Get("X-Proxy-Serial"))
	assert.Equal(t, "I'm a teapot", response.Status)
}