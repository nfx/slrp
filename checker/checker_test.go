package checker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pmux"
	"github.com/stretchr/testify/assert"
)

func TestFailure(t *testing.T) {
	c := NewChecker(&checkerShim{
		err: fmt.Errorf("fails"),
	})

	ctx := context.Background()
	_, err := c.Check(ctx, pmux.HttpProxy("127.0.0.1:1"))
	assert.EqualError(t, err, "fails")
}

func TestConfigurableChecker(t *testing.T) {
	client := http.DefaultClient
	c := configurableChecker{
		client: client,
	}
	err := c.Configure(app.Config{})
	assert.NoError(t, err)
	assert.Equal(t, time.Second*5, client.Timeout)
}

type checkerShim struct {
	http.Response
	err error
}

func (r checkerShim) Do(req *http.Request) (*http.Response, error) {
	return &r.Response, r.err
}

func (r checkerShim) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return nil, r.err
}

func body(x string) io.ReadCloser {
	return io.NopCloser(bytes.NewBufferString(x))
}

func TestTwoPassCheck(t *testing.T) {
	for i, tt := range []struct {
		firstBody, secondBody string
		firstErr, secondErr   error
		expectErr             string
	}{
		{
			firstBody: "..",
			secondErr: fmt.Errorf("nope"),
			expectErr: "second: nope",
		},
		{
			firstBody: "..",
			secondErr: temporary("second timeout"),
			expectErr: "second timeout",
		},
		{
			firstBody: "..",
			firstErr:  temporary("first timeout"),
			expectErr: "first timeout",
		},
		{
			firstBody: "..",
			firstErr:  fmt.Errorf("nope"),
			expectErr: "first: nope",
		},
		{
			firstBody:  "..",
			secondBody: "..",
		},
	} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			tp := twoPass{
				first: federated{
					&simple{
						ip:    "XYZ",
						valid: "..",
						client: checkerShim{
							Response: http.Response{
								Body:       body(tt.firstBody),
								StatusCode: 200,
							},
							err: tt.firstErr,
						},
					},
				},
				second: federated{
					&simple{
						ip:    "XYZ",
						valid: "..",
						client: checkerShim{
							Response: http.Response{
								Body:       body(tt.secondBody),
								StatusCode: 200,
							},
							err: tt.secondErr,
						},
					},
				},
			}
			_, err := tp.Check(context.Background(), pmux.HttpProxy("127.0.0.1:23"))
			if tt.expectErr != "" {
				assert.EqualError(t, err, tt.expectErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type failingReader string

func (f failingReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("%s", f)
}

func TestSimpleCheck(t *testing.T) {
	for i, tt := range []struct {
		body      io.ReadCloser
		err       error
		page      string
		expectErr string
		valid     string
		timeout   bool
	}{
		{
			page:      "😃://localhost",
			expectErr: `parse "😃://localhost": first path segment in URL cannot contain colon`,
		},
		{
			page:      "https://localhost",
			err:       temporary("slow"),
			expectErr: "slow",
			timeout:   true,
		},
		{
			page:      "https://localhost",
			body:      io.NopCloser(failingReader("😃")),
			expectErr: "😃",
		},
		{
			page:      "https://localhost",
			body:      io.NopCloser(failingReader("😃")),
			expectErr: "😃",
		},
		{
			page:      "https://localhost",
			body:      body(".. client does not have permission to get URL .."),
			expectErr: "google ratelimit",
			timeout:   true,
		},
		{
			page:      "https://localhost",
			body:      body(".. Cloudflare .."),
			expectErr: "cloudflare captcha",
			timeout:   true,
		},
		{
			page:      "https://localhost",
			body:      body(".. 255.0.0.1 .."),
			expectErr: "this IP address found",
		},
		{
			page:      "https://localhost",
			body:      body(".."),
			expectErr: "not ip: ..",
		},
		{
			page:      "https://localhost",
			body:      body(".."),
			valid:     "abc",
			expectErr: "no abc found: ..",
		},
		{
			page:      "https://localhost",
			body:      body(strings.Repeat("x", 514)),
			expectErr: fmt.Sprintf("not ip: %s (2b more)", strings.Repeat("x", 512)),
		},
	} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			s := &simple{
				ip:    "255.0.0.1",
				valid: tt.valid,
				page:  tt.page,
				client: checkerShim{
					Response: http.Response{
						Body:       tt.body,
						StatusCode: 200,
					},
					err: tt.err,
				},
			}
			_, err := s.Check(context.Background(), pmux.HttpProxy("127.0.0.1:23"))
			if tt.expectErr != "" {
				assert.EqualError(t, err, tt.expectErr)
				assert.Equal(t, tt.timeout, isTimeout(err), "is this error timeout?")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
