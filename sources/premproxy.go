package sources

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/nfx/slrp/pmux"

	"github.com/dop251/goja"
)

func init() {
	Sources = append(Sources, Source{
		ID:        5,
		Homepage:  "https://premproxy.com/list/",
		UrlPrefix: "https://premproxy.com/",
		Frequency: 6 * time.Hour,
		Feed:      premproxy,
	})
}

func deobfuscatePorts(script string) (map[string]string, error) {
	vm := goja.New()
	_, err := vm.RunString(`
	var document = null;
	var readyCb = null;
	var sets = {};
	function $(param) {
		this.ready = function(cb) {
			readyCb = cb;
		}
		this.html = function(value) {
			sets[param] = value;
		}
		return this
	}`)
	if err != nil {
		return nil, err
	}
	_, err = vm.RunString(script)
	if err != nil {
		return nil, err
	}
	_, err = vm.RunString(`readyCb()`)
	if err != nil {
		return nil, err
	}
	var sets map[string]string
	err = vm.ExportTo(vm.Get("sets"), &sets)
	return sets, err
}

func permproxyMapping(ctx context.Context, h *http.Client, html []byte, referer string) (map[string]string, error) {
	base := "https://premproxy.com"
	match := premproxyPortMappingScriptRE.FindSubmatch(html)
	if len(match) == 0 {
		return nil, skipRetry("cannot find script location")
	}
	scriptUrl := base + string(match[1])
	packedJS, _, err := req{
		URL: scriptUrl,
		Headers: map[string]string{
			"Referer": referer,
		},
	}.Do(ctx, h)
	if err != nil {
		return nil, err
	}
	// TODO: perhaps we should do "parent serial"?...
	return deobfuscatePorts(string(packedJS))
}

//<script src="/js-socks/7e65e.js">
var premproxyPortMappingScriptRE = regexp.MustCompile(`(?m)<script src="(/(js|js-socks)/[^\.]+.js)">`)
var premproxyObfuscatedAddrRE = regexp.MustCompile(`(?m)\d{1,3}.\d{1,3}.\d{1,3}.\d{1,3}\|[^"]+`)
var premproxyObfuscatedSocksAddrRE = regexp.MustCompile(`(?m)(\d{1,3}.\d{1,3}.\d{1,3}.\d{1,3}\|[^"]+).*(SOCKS[4|5])`)

func premproxyHttpPage(ctx context.Context, h *http.Client, url string) func() ([]pmux.Proxy, error) {
	return func() ([]pmux.Proxy, error) {
		found := []pmux.Proxy{}
		html, _, err := req{
			URL: url,
			Headers: map[string]string{
				"Referer": "https://premproxy.com/",
			},
		}.Do(ctx, h)
		if err != nil {
			return nil, err
		}
		mapping, err := permproxyMapping(ctx, h, html, url)
		if err != nil {
			return nil, err
		}
		for _, match := range premproxyObfuscatedAddrRE.FindAllString(string(html), -1) {
			split := strings.Split(match, "|")
			if len(split) != 2 {
				continue
			}
			port, ok := mapping["."+split[1]]
			if !ok {
				continue
			}
			addr := fmt.Sprintf("%s:%s", split[0], port)
			found = append(found, pmux.HttpsProxy(addr))
		}
		return found, nil
	}
}

func premproxySocksPage(ctx context.Context, h *http.Client, url string) func() ([]pmux.Proxy, error) {
	return func() ([]pmux.Proxy, error) {
		found := []pmux.Proxy{}
		html, serial, err := req{
			URL: url,
			Headers: map[string]string{
				"Referer": "https://premproxy.com/",
			},
		}.Do(ctx, h)
		if err != nil {
			return nil, err
		}
		mapping, err := permproxyMapping(ctx, h, html, url)
		if err != nil {
			return nil, skipRetry("cannot read mapping from %s in serial %s: %s", url, serial, err)
		}
		for _, match := range premproxyObfuscatedSocksAddrRE.FindAllStringSubmatch(string(html), -1) {
			split := strings.Split(match[1], "|")
			if len(split) != 2 {
				continue
			}
			port, ok := mapping["."+split[1]]
			if !ok {
				continue
			}
			addr := fmt.Sprintf("%s:%s", split[0], port)
			found = append(found, pmux.NewProxy(addr, strings.ToLower(match[2])))
		}
		return found, nil
	}
}

func premproxy(ctx context.Context, h *http.Client) Src {
	m := merged()
	list := "https://premproxy.com/list/%02d.htm"
	for i := 1; i <= 5; i++ {
		url := fmt.Sprintf(list, i)
		m.refresh(premproxyHttpPage(ctx, h, url))
	}
	list = "https://premproxy.com/socks-list/%02d.htm"
	for i := 1; i <= 13; i++ {
		url := fmt.Sprintf(list, i)
		m.refresh(premproxySocksPage(ctx, h, url))
	}
	return m
}
