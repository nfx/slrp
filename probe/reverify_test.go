package probe

import (
	"net/http"
	"testing"

	"github.com/nfx/slrp/ipinfo"
	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/ql/eval"
	"github.com/stretchr/testify/assert"
)

type fakeProbe []reVerify

func (f fakeProbe) Snapshot() internal {
	m := map[pmux.Proxy]reVerify{}
	for _, v := range f {
		m[v.Proxy] = v
	}
	return internal{
		LastReverified: m,
	}
}

func TestReverifyAPI(t *testing.T) {
	a := &reverifyDashboard{
		Probe: fakeProbe{
			{pmux.HttpProxy("127.0.0.2:2345"), 4, 5},
			{pmux.HttpProxy("127.0.0.3:4356"), 6, 7},
		},
		Lookup: ipinfo.NoopIpInfo{
			Country: "Zimbabwe",
		},
	}
	res, err := a.HttpGet(&http.Request{})
	assert.NoError(t, err)

	qr := res.(*eval.QueryResult[inReverify])
	assert.Len(t, qr.Records, 2)
}
