package history

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/nfx/slrp/pmux"
)

type roundTripper struct {
	history   *History
	transport http.RoundTripper
}

func justRead(r io.Reader) (buf []byte) {
	if r != nil {
		buf, _ = ioutil.ReadAll(r)
	}
	return
}

func (rt roundTripper) outBody(out *http.Response, err error) ([]byte, *http.Response) {
	if out == nil {
		return nil, &http.Response{
			StatusCode: 551,
			Status:     err.Error(),
			Header:     http.Header{},
		}
	}
	outBody := justRead(out.Body) // todo: check for leaked fds
	out.Body = ioutil.NopCloser(bytes.NewBuffer(outBody))
	return outBody, out
}

func (rt roundTripper) headersToMap(h http.Header) map[string]string {
	res := map[string]string{}
	for header := range h {
		res[header] = h.Get(header)
	}
	return res
}

func (rt roundTripper) popIntHeader(h http.Header, k string) int {
	v := h.Get(k)
	if v == "" {
		return 0
	}
	h.Del(k)
	i, _ := strconv.Atoi(v)
	return i
}

func (rt roundTripper) RoundTrip(in *http.Request) (*http.Response, error) {
	start := time.Now()
	// remove serial and attempt meta-headers before passing request to
	// other roundtripper, so that they don't leak to destination.
	serial := rt.popIntHeader(in.Header, "X-Proxy-Serial")
	attempt := rt.popIntHeader(in.Header, "X-Proxy-Attempt")
	// get proxy used for making the request
	proxy := pmux.GetProxyFromContext(in.Context())
	// perform actual HTTP round trip
	out, err := rt.transport.RoundTrip(in)
	// read read response body or fill it with just enough defaults
	outBody, out := rt.outBody(out, err)
	// record concise information about the request for debugging purposes
	rt.history.Record(Request{
		Serial:     serial,
		Attempt:    attempt,
		Ts:         time.Now(),
		Method:     in.Method,
		URL:        in.URL.String(),
		StatusCode: out.StatusCode,
		Status:     out.Status,
		Proxy:      proxy,
		InHeaders:  rt.headersToMap(in.Header),
		OutHeaders: rt.headersToMap(out.Header),
		InBody:     justRead(in.Body),
		OutBody:    outBody,
		Took:       time.Since(start),
	})
	return out, err
}
