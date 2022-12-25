package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeoNodeFixtures(t *testing.T) {
	pages := map[int][]geoNodeResult{
		1: {
			{
				IP:             "127.0.0.1",
				Port:           "12345",
				AnonymityLevel: "transparent",
				Protocols:      []string{"socks4"},
			},
			{
				IP:             "127.0.0.1",
				Port:           "12346",
				AnonymityLevel: "elite",
				Protocols:      []string{"socks5"},
			},
			{
				IP:             "127.0.0.1",
				Port:           "12347",
				AnonymityLevel: "anonymous",
				Protocols:      []string{"http"},
			},
		},
		2: {
			{
				IP:             "127.0.0.1",
				Port:           "12348",
				AnonymityLevel: "anonymous",
				Protocols:      []string{"http", "https"},
			},
			{
				IP:             "127.0.0.1",
				Port:           "12350",
				AnonymityLevel: "anonymous",
				Protocols:      []string{"socks4"},
			},
		},
		3: {},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var page int
		_, _ = fmt.Sscan(r.FormValue("page"), &page)
		w.WriteHeader(200)
		raw, _ := json.Marshal(geoNodeResultPage{
			Data: pages[page],
			Page: page,
		})
		w.Write(raw)
	}))
	defer server.Close()
	geoNodeURL = server.URL
	testSource(t, func(ctx context.Context) Src {
		return ByName("geonode.com").Feed(ctx, http.DefaultClient)
	}, 5)
}
