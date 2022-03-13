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

func newRegexPage(ctx context.Context, h *http.Client, url string,
	cb func(proxy string) pmux.Proxy) (found []pmux.Proxy, err error) {
	body, _, err := req{URL: url}.Do(ctx, h)
	if err != nil {
		return
	}
	return extractProxiesFromReader(ctx, url, body, cb)
}

func httpProxyRegexFeed(url string) func(context.Context, *http.Client) Src {
	return func(ctx context.Context, h *http.Client) Src {
		return gen(regexFeed(ctx, h, url, "http"))
	}
}

func regexFeed(ctx context.Context, h *http.Client, url, proto string) func() ([]pmux.Proxy, error) {
	return func() ([]pmux.Proxy, error) {
		return newRegexPage(ctx, h, url, func(proxy string) pmux.Proxy {
			return pmux.NewProxy(proxy, proto)
		})
	}
}

func regexFeedBase(ctx context.Context, h *http.Client, baseUrl string) func(path, proto string) func() ([]pmux.Proxy, error) {
	return func(path, proto string) func() ([]pmux.Proxy, error) {
		return regexFeed(ctx, h, baseUrl+path, proto)
	}
}

func extractProxiesFromReader(ctx context.Context, url string, body []byte,
	cb func(proxy string) pmux.Proxy) (found []pmux.Proxy, err error) {
	proxies := ipPortRegex.FindAll(body, -1)
	log := app.Log.From(ctx)
	if len(proxies) > 0 {
		log.Info().Int("count", len(proxies)).Str("url", url).Msg("found")
	}
	for _, proxy := range proxies {
		found = append(found, cb(string(proxy)))
	}
	return
}

func findLinksWithOn(ctx context.Context, h *http.Client, with string, page string) (links []string, err error) {
	body, serial, err := req{URL: page}.Do(ctx, h)
	if err != nil {
		return nil, err
	}
	document, err := goquery.NewDocumentFromReader(bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("%s / serial:%s", err, serial)
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
	URL          string
	Body         io.Reader
	SkipOnStatus int
	Headers      map[string]string
}

var blockers = []string{
	"captcha",
	"contentkeeper.net", // body onload submit, perhaps can be tuned?...
}

func (r req) Do(ctx context.Context, h *http.Client) ([]byte, string, error) {
	request, _ := http.NewRequestWithContext(ctx, "GET", r.URL, r.Body)
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
		if resp.StatusCode == 552 {
			// retry on proxy pool exhaustion
			continue
		}
		if err != nil {
			return nil, "", err
		}
	}
	serial := resp.Header.Get("X-Proxy-Serial")
	if resp.StatusCode >= 400 {
		return nil, serial, fmt.Errorf(resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if resp.Body != nil {
		resp.Body.Close()
	}
	if len(body) > 0 {
		for _, v := range blockers {
			// TODO: highlight in history?...
			if strings.Contains(strings.ToLower(string(body)), v) {
				return nil, serial, fmt.Errorf("found %s", v)
			}
		}
	}
	if r.SkipOnStatus > 0 && resp.StatusCode == r.SkipOnStatus {
		return nil, serial, skipRetry("skipping on %d. serial %s", resp.StatusCode, serial)
	}
	return body, serial, err
}

func newTablePage(ctx context.Context, h *http.Client, url string) (*htmltable.Page, error) {
	body, serial, err := req{URL: url}.Do(ctx, h)
	if err != nil {
		return nil, err
	}
	page, err := htmltable.New(ctx, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	if page.Len() == 0 {
		return nil, skipRetry("no tables found: %s / %s", url, serial)
	}
	return page, nil
}

func mustParseInt(value string) int {
	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0
	}
	return int(n)
}

var accept = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"
