package history

import (
	"net/http"
	"testing"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pmux"
	"github.com/stretchr/testify/assert"
)

func TestRecordNotFound(t *testing.T) {
	history, runtime := app.MockStartSpin(NewHistory())
	defer runtime.Stop()
	_, err := history.HttpGetByID("23456", &http.Request{})
	assert.EqualError(t, err, "request not found: 23456")
}

func TestRecord(t *testing.T) {
	history := NewHistory()
	runtime := app.Singletons{"_": history}.MockStart()
	defer runtime.Stop()

	go history.Record(Request{
		Proxy: pmux.HttpProxy("1.2.3.4:56789"),
	})
	// wait until request is recorded
	<-runtime["_"].Wait

	// and then start spinning to simplify testing
	runtime["_"].Spin()

	x, err := history.HttpGetByID("1", &http.Request{})
	assert.NoError(t, err)

	request, ok := x.(Request)
	assert.True(t, ok)
	assert.Equal(t, "http://1.2.3.4:56789", request.Proxy.String())

	x, err = history.HttpGet(&http.Request{})
	assert.NoError(t, err)
	fr := x.(filterResults)
	assert.Equal(t, 1, len(fr.Requests))
}
