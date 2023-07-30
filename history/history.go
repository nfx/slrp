package history

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/ql/eval"

	"github.com/yosssi/gohtml"
)

type filterResults struct {
	eval.QueryResult[filteredRequest]
	Err error `json:",omitempty"`
}

type filteredRequest struct {
	ID         int
	Serial     int
	Attempt    int
	Ts         time.Time
	Method     string
	URL        string
	StatusCode int
	Status     string
	Proxy      string
	Appeared   int
	Size       int
	Took       float64
}

//go:generate go run ../ql/generator/main.go Request
type Request struct {
	ID         int
	Serial     int
	Attempt    int `facet:"Attempt"`
	Ts         time.Time
	Method     string     `facet:"Method"`
	URL        string     `facet:"Host"`
	StatusCode int        `facet:"Status Code"`
	Status     string     `facet:"Status"`
	Proxy      pmux.Proxy `facet:"Proxy"`
	Appeared   int
	InHeaders  map[string]string
	OutHeaders map[string]string
	InBody     []byte
	OutBody    []byte
	Size       int
	Took       time.Duration
}

func (d RequestDataset) getHostname(record int) string {
	original := d[record].URL
	u, err := url.Parse(original)
	if err == nil {
		return u.Host
	} else {
		return original
	}
}

func (d RequestDataset) getStatusFacet(record int) string {
	split := strings.Split(d[record].Status, ": ")
	// perform error common suffix normalisation
	failure := split[len(split)-1]
	return failure
}

func (r Request) String() string {
	buf := []string{}

	buf = append(buf, fmt.Sprintf("%s %s %d (%s)", r.Method, r.URL, r.StatusCode, r.Status))
	buf = append(buf, fmt.Sprintf("* Serial: %d | Attempt: %d", r.Serial, r.Attempt))
	buf = append(buf, fmt.Sprintf("* Via: %s | Took: %s", r.Proxy, r.Took))
	for k, v := range r.InHeaders {
		buf = append(buf, fmt.Sprintf("> %s: %s", k, v))
	}
	for k, v := range r.OutHeaders {
		buf = append(buf, fmt.Sprintf("< %s: %s", k, v))
	}

	if len(r.OutBody) > 0 {
		pretty := gohtml.FormatWithLineNo(string(r.OutBody))
		buf = append(buf, pretty)
	}

	return strings.Join(buf, "\n")
}

type filter struct {
	Query string
	out   chan filterResults
}

type requestRequest struct {
	ID  int
	out chan Request
}

type History struct {
	requestRequest chan requestRequest
	filter         chan filter
	record         chan Request
	requests       RequestDataset
	appears        map[pmux.Proxy]int
	limit          int
}

func NewHistory() *History {
	return &History{
		requests:       RequestDataset{},
		requestRequest: make(chan requestRequest),
		filter:         make(chan filter),
		record:         make(chan Request, 128),
		appears:        map[pmux.Proxy]int{},
	}
}

func (h *History) Configure(c app.Config) error {
	h.limit = c.IntOr("limit", 1000)
	return nil
}

func (h *History) Start(ctx app.Context) {
	go h.main(ctx)
}

func (h *History) Wrap(transport http.RoundTripper) http.RoundTripper {
	return roundTripper{h, transport}
}

func (h *History) Record(r Request) {
	h.record <- r
}

func (h *History) HttpGet(r *http.Request) (interface{}, error) {
	res := h.sendFilter(r)
	return res, res.Err
}

func (h *History) HttpGetByID(id string, r *http.Request) (interface{}, error) {
	id_, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	d := h.get(id_)
	if d.ID == 0 {
		return nil, app.NotFound("request not found: " + id)
	}
	return d, nil
}

func (h *History) sendFilter(r *http.Request) filterResults {
	out := make(chan filterResults)
	defer close(out)
	h.filter <- filter{
		Query: r.FormValue("filter"),
		out:   out,
	}
	return <-out
}

func (h *History) get(id int) Request {
	out := make(chan Request)
	defer close(out)
	h.requestRequest <- requestRequest{
		ID:  id,
		out: out,
	}
	return <-out
}

func (h *History) main(ctx app.Context) {
	counter := 0
	for {
		select {
		case <-ctx.Done():
			return
		case r := <-h.record:
			// this may turn into partitioned data structure or index?..
			counter++
			r.ID = counter
			h.appears[r.Proxy] += 1
			r.Appeared = h.appears[r.Proxy]
			if h.limit > 0 && len(h.requests) == h.limit {
				h.requests = h.requests[1:]
			}
			h.requests = append(h.requests, r)
			ctx.Heartbeat()
		case r := <-h.requestRequest:
			var found bool
			for i := 0; i < len(h.requests); i++ {
				// this is very naive impl, replace with at least binary search.
				// keep in mind that serial is _nearly_ in order, but needs sorting
				if h.requests[i].ID != r.ID {
					continue
				}
				r.out <- h.requests[i]
				found = true
				break
			}
			if !found {
				r.out <- Request{}
			}
		case f := <-h.filter:
			f.out <- h.handleFilter(f)
		}
	}
}

func (h *History) handleFilter(f filter) filterResults {
	res, err := h.requests.Query(f.Query)
	if err != nil {
		return filterResults{
			Err: err,
		}
	}
	// re-create []filteredRequest from *eval.QueryResults
	out := filterResults{
		QueryResult: eval.QueryResult[filteredRequest]{
			Total:  res.Total,
			Facets: res.Facets,
		},
	}
	for _, v := range res.Records {
		out.QueryResult.Records = append(out.QueryResult.Records, filteredRequest{
			ID:         v.ID,
			Serial:     v.Serial,
			Attempt:    v.Attempt,
			Ts:         v.Ts,
			Method:     v.Method,
			URL:        v.URL,
			Status:     v.Status,
			StatusCode: v.StatusCode,
			Proxy:      v.Proxy.String(),
			Appeared:   h.appears[v.Proxy],
			Size:       len(v.OutBody),
			Took:       v.Took.Round(time.Second).Seconds(),
		})
	}
	return out
}
