package sources

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/nfx/slrp/pmux"
	"github.com/stretchr/testify/assert"
)

func dummy(proxy string) pmux.Proxy {
	return pmux.NewProxy("1.2.3.4:56", "http")
}

func TestNewRegexPageError(t *testing.T) {
	_, err := newRegexPage(context.Background(), nil, "", "..", dummy)
	assert.EqualError(t, err, "no http client (skip)")
}

func TestHttpProxyRegexFeedError(t *testing.T) {
	src := httpProxyRegexFeed("..", "..")(nil, nil)
	<-src.Generate(context.Background())
	err := src.Err()
	assert.EqualError(t, err, "no http client (skip)")
}

func TestExtractProxiesFromReaderNoProxies(t *testing.T) {
	_, err := extractProxiesFromReader(context.Background(), "..", []byte{}, dummy)
	assert.NoError(t, err)
}

func TestFindLinksWithOnError(t *testing.T) {
	_, err := findLinksWithOn(context.Background(), nil, "..", "..")
	assert.EqualError(t, err, "no http client (skip)")
}

type failingReader string

func (f failingReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("%s", f)
}

func Test_findLinksWithInBytes_FailingReader(t *testing.T) {
	_, err := findLinksWithInBytes(failingReader("nope"), 1, "..", "..")
	assert.EqualError(t, err, "nope serial=1")
}

func Test_findLinksWithInBytes_noLinksWith(t *testing.T) {
	links, err := findLinksWithInBytes(bytes.NewBufferString("<a>b</a>"), 1, "..", "..")
	assert.NoError(t, err)
	assert.Len(t, links, 0)
}

func Test_findLinksWithInBytes_some(t *testing.T) {
	links, err := findLinksWithInBytes(bytes.NewBufferString(`
	<b>lorem ipsum<a href="/amet/flag">first</a></b>
	<a href="/new">second</a>
	<div>
		<!-- some -->
		<i><a href="/amet/twe">second</a>
	</dir>
	`), 1, "amet", "https://localhost:23412")
	assert.NoError(t, err)
	assert.Equal(t, []string{
		"https://localhost:23412/amet/flag",
		"https://localhost:23412/amet/twe",
	}, links)
}

type staticResponseClient struct {
	http.Response
	err error
}

func (r staticResponseClient) Do(req *http.Request) (*http.Response, error) {
	return &r.Response, r.err
}

func TestReqDo_Err(t *testing.T) {
	_, _, err := req{}.Do(context.Background(), staticResponseClient{
		err: fmt.Errorf("nope"),
	})
	assert.EqualError(t, err, "nope")
}

func TestReqDo_NoBody(t *testing.T) {
	_, _, err := req{URL: ".."}.Do(context.Background(), staticResponseClient{
		Response: http.Response{},
	})
	assert.EqualError(t, err, "nil body: GET ..")
}

func TestReqDo_NoBody_PoolExhausted(t *testing.T) {
	_, _, err := req{URL: ".."}.Do(context.Background(), staticResponseClient{
		Response: http.Response{
			StatusCode: 552,
		},
	})
	assert.EqualError(t, err, "nil body: GET ..")
}

func TestReqDo_BlockerDetected(t *testing.T) {
	_, _, err := req{URL: ".."}.Do(context.Background(), staticResponseClient{
		Response: http.Response{
			Header: http.Header{
				"X-Proxy-Serial": []string{"123456"},
			},
			Body: io.NopCloser(bytes.NewBufferString(".. Cloudflare ..")),
		},
	})
	assert.EqualError(t, err, "found blocker serial=123456 marker=cloudflare")
}

func TestReqDo_SkipOnStatus(t *testing.T) {
	_, _, err := req{SkipOnStatus: 417}.Do(context.Background(), staticResponseClient{
		Response: http.Response{
			StatusCode: 417,
			Body:       io.NopCloser(bytes.NewBufferString("..")),
		},
	})
	assert.EqualError(t, err, "skip status serial=0 statusCode=417 (skip)")
}

func TestReqDo_ErrStatus(t *testing.T) {
	_, _, err := req{}.Do(context.Background(), staticResponseClient{
		Response: http.Response{
			StatusCode: 400,
			Status:     "Nope",
			Body:       io.NopCloser(bytes.NewBufferString("..")),
		},
	})
	assert.EqualError(t, err, "error status serial=0 code=400 status=Nope")
}

func TestReqDo_EmptyBody(t *testing.T) {
	_, _, err := req{}.Do(context.Background(), staticResponseClient{
		Response: http.Response{
			Body: io.NopCloser(bytes.NewBufferString("")),
		},
	})
	assert.EqualError(t, err, "empty body serial=0")
}

func TestReqDo_EmptyBodyValid(t *testing.T) {
	_, _, err := req{EmptyBodyValid: true}.Do(context.Background(), staticResponseClient{
		Response: http.Response{
			Body: io.NopCloser(bytes.NewBufferString("")),
		},
	})
	assert.NoError(t, err)
}

func TestReqDo_ExpectInResponse(t *testing.T) {
	_, _, err := req{ExpectInResponse: "b"}.Do(context.Background(), staticResponseClient{
		Response: http.Response{
			Body: io.NopCloser(bytes.NewBufferString("aaa")),
		},
	})
	assert.EqualError(t, err, "invalid response serial=0 expect=b")
}

func Test_newTablePage_Err(t *testing.T) {
	_, _, err := newTablePage(context.Background(), nil, "..", "..")
	assert.EqualError(t, err, "no http client (skip)")
}

func Test_newTablePage_NoTablesFound(t *testing.T) {
	_, _, err := newTablePage(context.Background(), staticResponseClient{
		Response: http.Response{
			Body: io.NopCloser(bytes.NewBufferString("aaa ..")),
		},
	}, "..", "..")
	assert.EqualError(t, err, "no tables found serial=0 url=.. (skip)")
}

func Test_mustParseInt(t *testing.T) {
	assert.Equal(t, 0, mustParseInt(".."))
}
