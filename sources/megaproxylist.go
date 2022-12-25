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
		ID:        69,
		Homepage:  "https://www.megaproxylist.net",
		Frequency: 24 * time.Hour,
		Seed:      true,
		Feed:      simpleGen(checkerProxy),
	})
}

type megaproxylistLine struct {
	address   string
	port      string
	country   string
	reability string
}

var megaproxylistUrl = fmt.Sprintf("https://www.megaproxylist.net/download/megaproxylist-csv-%s_SDACH.zip", time.Now().Format("20060102"))

// Scrapes https://www.megaproxylist.net
func Megaproxylist(ctx context.Context, h *http.Client) (found []pmux.Proxy, err error) {
	log.Info().Msg("Loading proxy checker database")

	resp, err := http.Get(megaproxylistUrl)
	if err != nil {
		return
	}
	if resp.Body == nil {
		return nil, fmt.Errorf("body is nil")
	}
	defer resp.Body.Close()
	csv_data := unzipInMemory(ctx, resp)
	r := csv.NewReader(bytes.NewBuffer(csv_data))
	r.Comma = ';'
	r.TrimLeadingSpace = true

	// trick to skip header
	if _, err := r.Read(); err != nil {
		panic(err)
	}

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		/*
			        if err != nil {
						log.Fatal(err)
					}
		*/
		fmt.Println(record)

		proxy := megaproxylistLine{
			address:   record[0],
			port:      record[1],
			country:   record[2],
			reability: record[3],
		}
		fmt.Println(fmt.Sprintf("%s:%s", proxy.address, proxy.port))
		found = append(found,
			pmux.NewProxy(fmt.Sprintf("%s:%s", proxy.address, proxy.port),
				"http"))
	}

	return
}
