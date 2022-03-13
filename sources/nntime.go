package sources

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/nfx/slrp/pmux"
)

func init() {
	Sources = append(Sources, Source{
		ID:        4,
		Homepage:  "http://nntime.com",
		Frequency: 6 * time.Hour,
		Feed:      newNetTimeNew,
	})
}

func newNetTimeNew(ctx context.Context, h *http.Client) Src {
	return merged().
		refresh(newNetTimePage(ctx, h, 1)).
		refresh(newNetTimePage(ctx, h, 2)).
		refresh(newNetTimePage(ctx, h, 3)).
		refresh(newNetTimePage(ctx, h, 4)).
		refresh(newNetTimePage(ctx, h, 5)).
		refresh(newNetTimePage(ctx, h, 6)).
		refresh(newNetTimePage(ctx, h, 7)).
		refresh(newNetTimePage(ctx, h, 8)).
		refresh(newNetTimePage(ctx, h, 9)).
		refresh(newNetTimePage(ctx, h, 10))
}

func newNetTimePage(ctx context.Context, h *http.Client, i int) func() ([]pmux.Proxy, error) {
	var mangedIPs = regexp.MustCompile(`(?m)>(?P<ip>\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}).*\":\"\+(.*)\)`)
	var mangles = regexp.MustCompile(`(?m)(?P<char>\w)=(?P<digit>\d);`)
	pattern := "http://nntime.com/proxy-updated-%02d.htm"
	return func() (found []pmux.Proxy, err error) {
		body, _, err := req{URL: fmt.Sprintf(pattern, i)}.Do(ctx, h)
		if err != nil {
			return
		}
		deMangle := map[string]string{}
		for _, perPageContext := range mangles.FindAllStringSubmatch(string(body), -1) {
			char, digit := perPageContext[1], perPageContext[2]
			deMangle[char] = digit
		}
		for _, mangledProxy := range mangedIPs.FindAllStringSubmatch(string(body), -1) {
			ip := mangledProxy[1]
			port := mangledProxy[2]
			for k, v := range deMangle {
				port = strings.ReplaceAll(port, k, v)
			}
			port = strings.ReplaceAll(port, "+", "")
			found = append(found, pmux.HttpProxy(fmt.Sprintf("%s:%s", ip, port)))
		}
		return
	}
}
