package sources

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func init() {
	Sources = append(Sources,
		github(12, "TheSpeedX/PROXY-List", map[string]string{
			"http":   "/master/http.txt",
			"socks4": "/master/socks4.txt",
			"socks5": "/master/socks5.txt",
		}, 3*time.Hour),
		github(17, "jetkai/proxy-list", map[string]string{
			"http":   "/main/online-proxies/txt/proxies-http.txt",
			"https":  "/main/online-proxies/txt/proxies-https.txt",
			"socks4": "/main/online-proxies/txt/proxies-socks4.txt",
			"socks5": "/main/online-proxies/txt/proxies-socks5.txt",
		}, 2*time.Hour),
		github(25, "almroot/proxylist", map[string]string{
			"http": "/master/list.txt",
		}, 1*time.Hour),
		github(26, "andigwandi/free-proxy", map[string]string{
			"http": "/main/proxy_list.txt",
		}, 2*time.Hour),
		github(27, "aslisk/proxyhttps", map[string]string{
			"http": "/main/https.txt",
		}, 24*time.Hour),
		github(28, "B4RC0DE-TM/proxy-list", map[string]string{
			"http":   "/main/HTTP.txt",
			"socks4": "/main/SOCKS4.txt",
			"socks5": "/main/SOCKS5.txt",
		}, 24*time.Hour),
		github(29, "BlackSnowDot/proxylist-update-every-minute", map[string]string{
			"http":   "/main/https.txt",
			"socks4": "/main/socks.txt",
		}, 15*time.Minute),
		github(30, "fahimscirex/proxybd", map[string]string{
			"http":   "/master/proxylist/http.txt",
			"socks4": "/master/proxylist/socks4.txt",
			"socks5": "/master/proxylist/socks5.txt",
		}, 4*time.Hour),
		github(31, "hanwayTech/free-proxy-list", map[string]string{
			"http":   "/main/https.txt",
			"socks4": "/main/socks4.txt",
			"socks5": "/main/socks5.txt",
		}, 1*time.Hour),
		github(32, "hendrikbgr/Free-Proxy-Repo", map[string]string{
			"http": "/master/proxy_list.txt",
		}, 24*time.Hour),
		github(33, "hookzof/socks5_list", map[string]string{
			"socks5": "/master/proxy.txt",
		}, 5*time.Minute),
		github(34, "HyperBeats/proxy-list", map[string]string{
			"http":   "/main/http.txt",
			"socks4": "/main/socks4.txt",
			"socks5": "/main/socks5.txt",
		}, 2*time.Hour),
		github(35, "manuGMG/proxy-365", map[string]string{
			"socks5": "/main/SOCKS5.txt",
		}, 1*time.Hour),
		github(36, "mertguvencli/http-proxy-list", map[string]string{
			"http": "/main/proxy-list/data.txt",
		}, 10*time.Minute),
		github(37, "miyukii-chan/proxy-list", map[string]string{
			"http":   "/master/proxies/http.txt",
			"socks4": "/master/proxies/socks4.txt",
			"socks5": "/master/proxies/socks5.txt",
		}, 24*time.Hour),
		github(38, "mmpx12/proxy-list", map[string]string{
			"http":   "/master/https.txt",
			"socks4": "/master/socks4.txt",
			"socks5": "/master/socks5.txt",
		}, 1*time.Hour),
		github(39, "monosans/proxy-list", map[string]string{
			"http":   "/main/proxies/http.txt",
			"socks4": "/main/proxies/socks4.txt",
			"socks5": "/main/proxies/socks5.txt",
		}, 30*time.Minute),
		github(40, "ObcbO/getproxy", map[string]string{
			"http":   "/master/https.txt",
			"socks4": "/master/socks4.txt",
			"socks5": "/master/socks5.txt",
		}, 6*time.Hour),
		github(41, "officialputuid/KangProxy", map[string]string{
			"http":   "/KangProxy/https/https.txt",
			"socks4": "/KangProxy/socks4/socks4.txt",
			"socks5": "/KangProxy/socks5/socks5.txt",
		}, 2*time.Hour),
		github(42, "proxy4parsing/proxy-list", map[string]string{
			"http": "/main/http.txt",
		}, 15*time.Minute),
		// https://github.com/proxylist-to/proxy-list - get not from github...
		// github(43, "proxylist-to/proxy-list", map[string]string{
		// 	"http":   "/main/http.txt",
		// 	"socks4": "/main/socks4.txt",
		// 	"socks5": "/main/socks5.txt",
		// }, 3*time.Hour),
		github(44, "rdavydov/proxy-list", map[string]string{
			"http":   "/main/proxies/http.txt",
			"socks4": "/main/proxies/socks4.txt",
			"socks5": "/main/proxies/socks5.txt",
		}, 30*time.Minute),
		github(45, "ReCaree/proxy-scrapper", map[string]string{
			"http":   "/master/proxy/http-removed.txt",
			"socks4": "/master/proxy/socks4-removed.txt",
			"socks5": "/master/proxy/socks5-removed.txt",
		}, 24*time.Hour),
		github(46, "roosterkid/openproxylist", map[string]string{
			"http":   "/main/HTTPS_RAW.txt",
			"socks4": "/main/SOCKS4_RAW.txt",
			"socks5": "/main/SOCKS5_RAW.txt",
		}, 30*time.Minute),
		github(49, "saschazesiger/Free-Proxies", map[string]string{
			"http":   "/master/proxies/http.txt",
			"socks4": "/master/proxies/socks4.txt",
			"socks5": "/master/proxies/socks5.txt",
		}, 1*time.Hour),
		github(50, "ShiftyTR/Proxy-List", map[string]string{
			"http":   "/master/https.txt",
			"socks4": "/master/socks4.txt",
			"socks5": "/master/socks5.txt",
		}, 10*time.Minute),
		github(51, "UptimerBot/proxy-list", map[string]string{
			"http":   "/main/proxies/http.txt",
			"socks4": "/main/proxies/socks4.txt",
			"socks5": "/main/proxies/socks5.txt",
		}, 1*time.Hour),
		github(52, "yemixzy/proxy-list", map[string]string{
			"http": "/main/proxy-list/data.txt",
		}, 24*time.Hour),
		github(53, "Zaeem20/FREE_PROXIES_LIST", map[string]string{
			"http":   "/master/https.txt",
			"socks4": "/master/socks4.txt",
			"socks5": "/master/socks5.txt",
		}, 1*time.Hour),
		github(54, "zevtyardt/proxy-list", map[string]string{
			"http":   "/main/http.txt",
			"socks4": "/main/socks4.txt",
			"socks5": "/main/socks5.txt",
		}, 2*time.Hour),
	)
}

func github(id int, repo string, files map[string]string, freq time.Duration) Source {
	ownerSplit := strings.Split(repo, "/")
	return Source{
		ID:        id,
		name:      ownerSplit[0],
		Homepage:  fmt.Sprintf("https://github.com/%s", repo),
		UrlPrefix: fmt.Sprintf("https://raw.githubusercontent.com/%s", repo),
		Frequency: freq,
		Seed:      true,
		Feed: func(ctx context.Context, h *http.Client) Src {
			f := regexFeedBase(ctx, h, fmt.Sprintf("https://raw.githubusercontent.com/%s", repo), ":")
			m := merged()
			for t, loc := range files {
				m.refresh(f(loc, t))
			}
			return m
		},
	}
}
