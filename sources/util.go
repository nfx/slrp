package sources

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pmux"

	"github.com/nfx/slrp/htmltable"

	"github.com/PuerkitoBio/goquery"
)

var countries = []string{
	"AL", "AM", "AR", "AT", "AU", "BA", "BD", "BG", "BO", "BR", "BY", "CA", "CL",
	"CM", "CO", "CZ", "EC", "EG", "ES", "FR", "GB", "GE", "GR", "GT", "HN", "HU",
	"ID", "IN", "IT", "JP", "KE", "KG", "KH", "KR", "KZ", "LT", "LV", "MD", "MM",
	"MN", "MU", "MW", "MX", "MY", "NO", "NP", "PE", "PH", "PK", "PL", "PY", "RO",
	"RS", "RU", "SC", "SE", "SG", "SK", "TH", "TR", "TZ", "UA", "UG", "US", "VE",
	"VN", "ZA",
}

var ipPortRegex = regexp.MustCompile(`(?m)\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d{2,5}`)

func newRegexPage(ctx context.Context, h *http.Client, url, expect string,
	cb func(proxy string) pmux.Proxy) (found []pmux.Proxy, err error) {
	body, serial, err := req{URL: url, ExpectInResponse: expect}.Do(ctx, h)
	if err != nil {
		return
	}
	ctx = app.Log.WithInt(ctx, "serial", serial)
	return extractProxiesFromReader(ctx, url, body, cb)
}

func httpProxyRegexFeed(url, expect string) func(context.Context, *http.Client) Src {
	return func(ctx context.Context, h *http.Client) Src {
		return gen(regexFeed(ctx, h, url, "http", expect))
	}
}

func regexFeed(ctx context.Context, h *http.Client, url, proto, expect string) func() ([]pmux.Proxy, error) {
	return func() ([]pmux.Proxy, error) {
		return newRegexPage(ctx, h, url, expect, func(proxy string) pmux.Proxy {
			return pmux.NewProxy(proxy, proto)
		})
	}
}

func regexFeedBase(ctx context.Context, h *http.Client, baseUrl, expect string) func(path, proto string) func() ([]pmux.Proxy, error) {
	return func(path, proto string) func() ([]pmux.Proxy, error) {
		return regexFeed(ctx, h, baseUrl+path, proto, expect)
	}
}

func extractProxiesFromReader(ctx context.Context, url string, body []byte,
	cb func(proxy string) pmux.Proxy) (found []pmux.Proxy, err error) {
	proxies := ipPortRegex.FindAll(body, -1)
	log := app.Log.From(ctx)
	if len(proxies) > 0 {
		log.Info().Int("count", len(proxies)).Str("url", url).Msg("found")
	} else {
		log.Info().Str("url", url).Msg("no proxies found")
	}
	for _, proxy := range proxies {
		found = append(found, cb(string(proxy)))
	}
	return
}

func findLinksWithOn(ctx context.Context, h *http.Client, with, page string) (links []string, err error) {
	body, serial, err := req{URL: page, ExpectInResponse: with}.Do(ctx, h)
	if err != nil {
		return nil, err
	}
	document, err := goquery.NewDocumentFromReader(bytes.NewBuffer(body))
	if err != nil {
		return nil, wrapError(err, intEC{"serial", serial})
	}
	document.Find("a").Each(func(i int, s *goquery.Selection) {
		href := s.AttrOr("href", "#")
		if !strings.Contains(href, with) {
			return
		}
		if href[0] == '/' {
			url, _ := url.Parse(page)
			// we skip the username/password for now
			href = fmt.Sprintf("%s://%s%s", url.Scheme, url.Host, href)
		}
		links = append(links, href)
	})
	return
}

type req struct {
	URL              string
	RequestBody      io.Reader
	ExpectInResponse string
	EmptyBodyValid   bool
	SkipOnStatus     int
	Headers          map[string]string
}

var blockers = []string{
	//"captcha", different...
	"Cloudflare",
	"contentkeeper.net", // body onload submit, perhaps can be tuned?...
}

func (r req) Do(ctx context.Context, h *http.Client) ([]byte, int, error) {
	request, _ := http.NewRequestWithContext(ctx, "GET", r.URL, r.RequestBody)
	if r.Headers != nil {
		for k, v := range r.Headers {
			request.Header.Set(k, v)
		}
	}
	attempt := 0
	var err error
	var resp *http.Response
	for attempt < 10 {
		attempt++
		resp, err = h.Do(request)
		if resp != nil && resp.StatusCode == 552 {
			// retry on proxy pool exhaustion
			continue
		}
		if err != nil {
			// though, serial might just work...
			return nil, 0, err
		}
		break
	}
	serial, err := strconv.Atoi(resp.Header.Get("X-Proxy-Serial"))
	if err != nil {
		serial = 0
	}
	body, err := io.ReadAll(resp.Body)
	if resp.Body != nil {
		resp.Body.Close()
	}
	if len(body) > 0 {
		for _, v := range blockers {
			// TODO: highlight in history?...
			if strings.Contains(strings.ToLower(string(body)), v) {
				return nil, serial, newErr("found blocker",
					intEC{"serial", serial}, strEC{"marker", v})
			}
		}
	}
	if r.SkipOnStatus > 0 && resp.StatusCode == r.SkipOnStatus {
		return nil, serial, skipError("skip status",
			intEC{"serial", serial}, intEC{"statusCode", resp.StatusCode})
	}
	if resp.StatusCode >= 400 {
		return nil, serial, newErr("error status",
			intEC{"serial", serial}, strEC{"status", resp.Status})
	}
	if len(body) == 0 && !r.EmptyBodyValid {
		return nil, serial, newErr("empty body",
			intEC{"serial", serial})
	}
	if r.ExpectInResponse != "" && !strings.Contains(string(body), r.ExpectInResponse) {
		return body, serial, newErr("invalid response",
			intEC{"serial", serial}, strEC{"expect", r.ExpectInResponse})
	}
	return body, serial, err
}

func newTablePage(ctx context.Context, h *http.Client, url, expect string) (*htmltable.Page, int, error) {
	body, serial, err := req{URL: url, ExpectInResponse: expect}.Do(ctx, h)
	if err != nil {
		return nil, serial, err
	}
	ctx = app.Log.WithInt(ctx, "serial", serial)
	page, err := htmltable.New(ctx, bytes.NewBuffer(body))
	if err != nil {
		return nil, serial, err
	}
	if page.Len() == 0 {
		return nil, serial, skipError("no tables found",
			intEC{"serial", serial}, strEC{"url", url})
	}
	return page, serial, nil
}

func mustParseInt(value string) int {
	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0
	}
	return int(n)
}

var accept = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"
