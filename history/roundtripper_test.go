package history

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/nfx/slrp/app"
	"github.com/stretchr/testify/assert"
)

func TestRoundTripper(t *testing.T) {
	hist, runtime := app.MockStartSpin(NewHistory())
	defer runtime.Stop()

	http.DefaultTransport = hist.Wrap(http.DefaultTransport)
	resp, err := http.Get("http://httpbin.org/get")
	assert.NoError(t, err)

	assert.Equal(t, 200, resp.StatusCode)

	res, err := hist.HttpGetByID("1", nil)
	assert.NoError(t, err)
	req := res.(Request)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "http://httpbin.org/get", req.URL)
}

type dummyTransport http.Response

func (et dummyTransport) RoundTrip(in *http.Request) (*http.Response, error) {
	if et.StatusCode == -1 {
		return nil, fmt.Errorf(et.Status)
	}
	x := http.Response(et)
	return &x, nil
}

func TestRoundTripperCornerCases(t *testing.T) {
	hist, runtime := app.MockStartSpin(NewHistory())
	defer runtime.Stop()

	roundTripper{hist, dummyTransport(http.Response{
		StatusCode: -1,
		Status:     "fail",
	})}.RoundTrip(&http.Request{
		Header: http.Header{
			"X-Proxy-Serial":  []string{"213"},
			"X-Proxy-Attempt": []string{"12"},
			"Abc":             []string{"nothing"},
		},
		Method: "GET",
		URL: &url.URL{
			Scheme: "http",
			Host:   "localhost",
		},
	})

	http.DefaultTransport = hist.Wrap(http.DefaultTransport)
	resp, err := http.Get("http://httpbin.org/get")
	assert.NoError(t, err)

	assert.Equal(t, 200, resp.StatusCode)

	res, err := hist.HttpGetByID("1", nil)
	assert.NoError(t, err)
	req := res.(Request)
	assert.Equal(t, "fail", req.Status)
	assert.Equal(t, 551, req.StatusCode)
	assert.Equal(t, 213, req.Serial)
	assert.Equal(t, 12, req.Attempt)
	assert.Equal(t, "", req.InHeaders["X-Proxy-Serial"])
	assert.Equal(t, "", req.InHeaders["X-Proxy-Attempt"])
	assert.Equal(t, "nothing", req.InHeaders["Abc"])
}
