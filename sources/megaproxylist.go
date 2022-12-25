package sources

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nfx/slrp/pmux"
	"github.com/rs/zerolog/log"
)

func init() {
	Sources = append(Sources, Source{
		ID:        65,
		Homepage:  "https://www.megaproxylist.net",
		Frequency: 24 * time.Hour,
		Seed:      true,
		Feed:      simpleGen(Megaproxylist),
	})
}

var megaproxylistUrl = fmt.Sprintf("https://www.megaproxylist.net/download/megaproxylist-csv-%s_SDACH.zip", time.Now().Format("20060102"))

// Scrapes https://www.megaproxylist.net
func Megaproxylist(ctx context.Context, h *http.Client) (found []pmux.Proxy, err error) {
	log.Info().Msg("Loading proxy Megaproxy database")

	resp, err := h.Get(megaproxylistUrl)
	if err != nil {
        log.Error().Msg(err.Error())
		return
	}
	if resp.Body == nil {
		log.Error().Msg("Empty body")
		return found, nil
	}
	defer resp.Body.Close()
	csvData := unzipInMemory(ctx, resp)
	r := csv.NewReader(bytes.NewBuffer(csvData))
	r.Comma = ';'
	r.TrimLeadingSpace = true
    // trick to skip header
	if _, err := r.Read(); err != nil {
		return found, nil
	}

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}

		found = append(found,
			pmux.NewProxy(fmt.Sprintf("%s:%s", record[0], record[1]),
				"http"))
	}

	return found, nil
}
