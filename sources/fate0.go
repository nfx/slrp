package sources

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nfx/slrp/pmux"
)

var fateZeroURL = "https://raw.githubusercontent.com/fate0/proxylist/master/proxy.list"

func init() {
	Sources = append(Sources, Source{
		ID:        22,
		name:      "fate0",
		Homepage:  "https://github.com/fate0/proxylist",
		Frequency: 15 * time.Minute,
		Feed:      simpleGen(fate0),
	})
}

type fate0line struct {
	Host         string  `json:"host"`
	Port         int     `json:"port"`
	Type         string  `json:"type"`
	Anonymity    string  `json:"anonymity"`
	ResponseTime float64 `json:"response_time"`
}

func fate0(ctx context.Context, h *http.Client) (found []pmux.Proxy, err error) {
	// this list is made by https://github.com/fate0/getproxy script,
	// which overlaps with the majority of existing sources,
	// but refreshed every 15 minutes.
	resp, err := http.Get(fateZeroURL)
	if err != nil {
		return
	}
	if resp.Body == nil {
		return nil, fmt.Errorf("body is nil")
	}
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var line fate0line
		err = json.Unmarshal(scanner.Bytes(), &line)
		if err != nil {
			return
		}
		if line.Anonymity == "transparent" {
			continue
		}
		found = append(found, pmux.NewProxy(
			fmt.Sprintf("%s:%d", line.Host, line.Port),
			line.Type))
	}
	return
}
