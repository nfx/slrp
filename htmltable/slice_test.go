package htmltable

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type nice struct {
	C string `header:"c"`
	D string `header:"d"`
}

func TestNewChannelFromString(t *testing.T) {
	out, err := NewSliceFromString[nice](fixture)
	assert.NoError(t, err)
	assert.Equal(t, []nice{
		{"2", "5"},
		{"4", "6"},
	}, out)
}

type Ticker struct {
	Symbol   string `header:"Symbol"`
	Security string `header:"Security"`
	CIK      string `header:"CIK"`
}

func TestNewChannelFromUrl(t *testing.T) {
	out, err := NewSliceFromURL[Ticker]("https://en.wikipedia.org/wiki/List_of_S%26P_500_companies")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(out), 500)
}

func TestNewChannelFromUrl_Fails(t *testing.T) {
	_, err := NewSliceFromURL[Ticker]("https://127.0.0.1")
	assert.EqualError(t, err, "Get \"https://127.0.0.1\": dial tcp 127.0.0.1:443: connect: connection refused")
}

func TestNewChannelFromUrl_NoTables(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()
	_, err := NewSliceFromURL[Ticker](server.URL)
	assert.EqualError(t, err, "cannot find table with columns: Symbol, Security, CIK")
}

func TestNewChannelInvalidTypes(t *testing.T) {
	type exotic struct {
		A string  `header:""`
		C float32 `header:"c"`
	}
	_, err := NewSliceFromString[exotic](fixture)
	assert.EqualError(t, err, "only strings are supported, C is float32")
}
