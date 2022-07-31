package checker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pmux"
	"github.com/stretchr/testify/assert"
)

func TestFailure(t *testing.T) {
	var proxy pmux.Proxy
	defer pmux.SetupHttpProxy(&proxy)()
	c := NewChecker()

	ctx := context.Background()
	_, err := c.Check(ctx, proxy)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConfigurableChecker(t *testing.T) {
	c := configurableChecker{
		client: http.DefaultClient,
	}
	err := c.Configure(app.Config{})
	assert.NoError(t, err)
	assert.Equal(t, "simple", c.strategy)
	assert.Equal(t, time.Second*5, c.client.Timeout)
}

type staticResponseClient struct {
	http.Response
	err error
}

func (r staticResponseClient) Do(req *http.Request) (*http.Response, error) {
	return &r.Response, r.err
}

func TestTwoPassCheck(t *testing.T) {
	tp := twoPass{
		first: federated{
			&simple{
				client: staticResponseClient{
					Response: http.Response{
						Body:       io.NopCloser(bytes.NewBufferString(".. Cloudflare ..")),
						StatusCode: 200,
					},
				},
			},
		},
		second: federated{
			&simple{
				client: staticResponseClient{
					err: fmt.Errorf("nope"),
				},
			},
		},
	}
	_, err := tp.Check(context.Background(), pmux.HttpProxy("127.0.0.1:23"))
	assert.EqualError(t, err, "cloudflare captcha")
}
