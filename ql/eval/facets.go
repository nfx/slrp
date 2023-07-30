package eval

import (
	"fmt"
	"math"
	"regexp"
	"time"

	"golang.org/x/exp/slices"
)

type Card struct {
	Name   string
	Value  int
	Filter string
}

type Facet struct {
	Name string
	Top  []Card
}

type FacetRetriever interface {
	Bucket() bucket
}

type FacetRetrievers[T any] []FacetRetriever

func (f FacetRetrievers[T]) Facets(result []T, topN int) (facets []Facet) {
	for _, retriever := range f {
		bucket := retriever.Bucket()
		for i := range result {
			bucket.Consume(i)
		}
		facet := bucket.Facet(topN)
		if len(facet.Top) == 0 {
			continue
		}
		facets = append(facets, facet)
	}
	return
}

type bucket interface {
	Consume(i int)
	Facet(n int) Facet
}

type NumberRanges struct {
	Getter   func(int) float64
	Field    string
	Name     string
	Duration bool
	Size     bool
	TimeAgo  bool
}

func (s NumberRanges) Bucket() bucket {
	return &numberRanges{NumberRanges: s}
}

type numberRanges struct {
	NumberRanges
	data []float64
}

func (s *numberRanges) Consume(i int) {
	s.data = append(s.data, s.Getter(i))
}

func (s *numberRanges) readableSize(bytes float64) string {
	b := int(bytes)
	if bytes < 1024 {
		return fmt.Sprintf("%db", b)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%dkb", b/1024)
	}
	return fmt.Sprintf("%dmb", b/1024/1024)
}

func (s *numberRanges) readableSince(t float64) string {
	x := time.Unix(int64(t), 0)
	day := 24 * time.Hour
	week := 7 * day
	month := 30*day + 12*time.Hour
	dur := time.Since(x)
	if dur < time.Minute {
		return fmt.Sprintf("%d seconds", int(dur/time.Second))
	} else if dur < time.Hour {
		return fmt.Sprintf("%d minutes", int(dur/time.Minute))
	} else if dur < day {
		return fmt.Sprintf("%d hours", int(dur/time.Hour))
	} else if dur < week {
		return fmt.Sprintf("%d days", int(dur/day))
	} else if dur < month {
		return fmt.Sprintf("%d weeks", int(dur/week))
	} else {
		return fmt.Sprintf("%d months", int(dur/month))
	}
}

func (s *numberRanges) Facet(n int) Facet {
	if n > 5 {
		// difficult to read more than 5 numeric ranges
		n = 5
	}
	min, max := math.MaxFloat64, math.SmallestNonzeroFloat64
	for _, v := range s.data {
		if v > max {
			max = v
		}
		if v < min {
			min = v
		}
	}

	type bin struct {
		start, end float64
		min, max   float64
		count      int
	}
	ranges := make([]bin, n)

	start := min
	rangeSize := (max - min) / float64(n)
	for i := 0; i < n; i++ {
		current := bin{
			start: start,
			end:   math.Ceil(start + rangeSize),
			min:   max,
		}
		// this triggers SIMD, right?..
		for _, v := range s.data {
			if v >= current.start && v <= current.end {
				if v > current.max {
					current.max = v
				}
				if v < current.min {
					current.min = v
				}
				current.count++
			}
		}
		start = current.end + 1
		ranges[i] = current
	}

	cards := []Card{}
	for _, r := range ranges {
		if r.count <= 1 {
			continue
		}
		start, end := int(r.min), int(r.max)
		name := fmt.Sprintf("%d .. %d", start, end)
		if s.TimeAgo {
			a, b := s.readableSince(r.min), s.readableSince(r.max)
			if a == b {
				name = a + " ago"
			} else {
				name = fmt.Sprintf("%s .. %s", a, b)
			}
		} else if s.Size {
			a, b := s.readableSize(r.min), s.readableSize(r.max)
			if a == b {
				name = a
			} else {
				name = fmt.Sprintf("%s .. %s", a, b)
			}
		} else if s.Duration {
			a, b := time.Duration(r.min), time.Duration(r.max)
			name = fmt.Sprintf("%s .. %s", a.Round(time.Millisecond).String(), b.Round(time.Millisecond).String())
		}
		cards = append(cards, Card{
			Name:  name,
			Value: r.count,
			// TODO: support lte/gte in the query engine
			Filter: fmt.Sprintf("%s > %d AND %s < %d", s.Field, start, s.Field, end),
		})
	}
	slices.SortStableFunc(cards, func(a, b Card) bool {
		if a.Value != b.Value {
			return a.Value > b.Value
		}
		return a.Name > b.Name
	})
	return Facet{
		Name: s.Name,
		Top:  cards,
	}
}

type StringFacet struct {
	Getter   func(int) string
	Field    string
	Name     string
	Contains bool
}

func (s StringFacet) Bucket() bucket {
	return &stringSummary{s, map[string]int{}}
}

type stringSummary struct {
	StringFacet
	summary map[string]int
}

func (s *stringSummary) Consume(i int) {
	s.summary[s.Getter(i)] += 1
}

var isAlpha = regexp.MustCompile(`^\w+$`)

func (s *stringSummary) Facet(n int) Facet {
	cards := []Card{}
	for name, cnt := range s.summary {
		if name == "" {
			name = "n/a"
		}
		if cnt < 2 {
			// TODO: perhaps match with N
			continue
		}
		q := name
		if !isAlpha.MatchString(q) {
			q = fmt.Sprintf(`"%s"`, q)
		}
		query := fmt.Sprintf("%s:%s", s.Field, q)
		if s.Contains {
			query = fmt.Sprintf("%s ~ %s", s.Field, q)
		}
		cards = append(cards, Card{
			Name:   name,
			Value:  cnt,
			Filter: query,
		})
	}
	slices.SortStableFunc(cards, func(a, b Card) bool {
		if a.Value != b.Value {
			return a.Value > b.Value
		}
		return a.Name > b.Name
	})
	min := n
	cardsLen := len(cards)
	if cardsLen < min {
		min = cardsLen
	}
	return Facet{
		Name: s.Name,
		Top:  cards[0:min],
	}
}
