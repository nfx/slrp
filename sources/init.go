package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Source struct {
	ID        int
	name      string
	Frequency time.Duration

	// Seed sources use http.DefaultClient to retrieve data,
	// all other sources use proxy pool to fetch pages.
	Seed bool

	Session      bool
	Homepage     string
	UrlPrefix    string
	expectString string

	Feed func(context.Context, *http.Client) Src
}

func (s Source) Name() string {
	if s.name != "" {
		return s.name
	}
	if s.Homepage != "" {
		page, err := url.Parse(s.Homepage)
		if err != nil {
			return fmt.Sprintf("src:%d", s.ID)
		}
		return strings.TrimPrefix(page.Host, "www.")
	}
	return fmt.Sprintf("src:%d", s.ID)
}

var Sources = []Source{}

func ByID(id int) Source {
	for _, s := range Sources {
		if s.ID != id {
			continue
		}
		return s
	}
	return Source{
		name: "unknown",
	}
}

func ByName(name string) Source {
	for _, s := range Sources {
		if s.Name() != name {
			continue
		}
		return s
	}
	return Source{
		name: "unknown",
	}
}
