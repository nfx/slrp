package sources

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pmux"

	"github.com/corpix/uarand"
)

// https://geekflare.com/best-rotating-proxy/

func init() {
	Sources = append(Sources,
		Source{
			ID:        14,
			Homepage:  "http://spys.one",
			Frequency: 3 * time.Hour,
			Feed:      simpleGen(spysOne),
		},
		Source{
			ID:        15,
			name:      "spys.me",
			Homepage:  "https://github.com/clarketm/proxy-list/",
			UrlPrefix: "https://raw.githubusercontent.com/clarketm/proxy-list",
			Frequency: 6 * time.Hour,
			Seed:      true,
			Feed:      httpProxyRegexFeed("https://raw.githubusercontent.com/clarketm/proxy-list/master/proxy-list-raw.txt"),
		})
}

func spysOne(ctx context.Context, h *http.Client) (found []pmux.Proxy, err error) {
	var xx0regex = regexp.MustCompile(`([a-z0-9]{32})`)
	ua := uarand.GetRandom()
	body, serial, err := req{
		URL: "https://spys.one/en/free-proxy-list/",
		Headers: map[string]string{
			"User-Agent":      ua,
			"Accept":          accept,
			"Accept-Language": "en-US,en;q=0.5",
		},
	}.Do(ctx, h)
	if err != nil {
		return nil, err
	}
	xx0 := xx0regex.FindString(string(body))
	if xx0 == "" {
		return nil, fmt.Errorf("cannot find xx0. serial: %s", serial)
	}
	for proxyType, xf5 := range map[string]string{"http": "1", "socks5": "2"} {
		for _, xf1 := range []string{"3", "4"} { // ANM, HIA
			form := url.Values{}
			form.Set("xx0", xx0)
			form.Set("xpp", "5")
			form.Set("xf1", xf1)
			form.Set("tldc", "0")
			form.Set("xf2", "0")
			form.Set("xf3", "0")
			form.Set("xf4", "0")
			form.Set("xf5", xf5)
			sleep := rand.Intn(15) // we have to be gently creative here
			time.Sleep(time.Duration(sleep) * time.Second)
			new, err := spysOnePage(ctx, h, ua, form, proxyType)
			if err != nil {
				return nil, err
			}
			found = append(found, new...)
		}
	}
	return found, err
}

func spysOnePage(ctx context.Context, h *http.Client, ua string, form url.Values, proxyType string) (found []pmux.Proxy, err error) {
	var mangles = regexp.MustCompile(`[>;]{1}(?P<char>[a-z\d]{4,})=(?P<num>[a-z\d\^]+)`)
	var mangled = regexp.MustCompile(`(?m)<script [^\+]+(?P<js_port_code>(?:\+\([a-z0-9^+]+\))+)\)<\/script>`)
	page := "https://spys.one/en/free-proxy-list/"
	body, _, err := req{
		URL:  page,
		Body: strings.NewReader(form.Encode()),
		Headers: map[string]string{
			"Accept":          accept,
			"Accept-Language": "en-US,en;q=0.5",
			"Content-Type":    "application/x-www-form-urlencoded",
			"Referer":         page,
			"User-Agent":      ua,
		},
	}.Do(ctx, h)
	if err != nil {
		return
	}
	deMangle := map[string]int{}
	for _, perPageContext := range mangles.FindAllStringSubmatch(string(body), -1) {
		char, digit := perPageContext[1], perPageContext[2]
		if strings.Contains(digit, "^") {
			sep := strings.Split(digit, "^")
			tmp := mustParseInt(sep[0]) ^ deMangle[sep[1]]
			digit = fmt.Sprintf("%d", tmp)
		}
		deMangle[char] = mustParseInt(digit)
	}
	i := 0
	body = mangled.ReplaceAllFunc(body, func(s []byte) []byte {
		port := ":"
		parts := strings.Split(string(s), "+")
		for i := 1; i < len(parts); i++ {
			p := strings.ReplaceAll(parts[i], ")</script>", "")
			p = strings.Trim(p, "()")
			labels := strings.Split(p, "^")
			// TODO: add more security otherwise this'll panic like hell
			digit := deMangle[labels[0]] ^ deMangle[labels[1]]
			port += fmt.Sprintf("%d", digit)
		}
		i++
		return []byte(port)
	})
	log := app.Log.From(ctx)
	log.Info().Int("count", i).Msg("found")
	return extractProxiesFromReader(ctx, page+"#"+proxyType, body, func(proxy string) pmux.Proxy {
		return pmux.NewProxy(proxy, proxyType)
	})
}
