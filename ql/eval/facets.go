package eval

import (
	"fmt"
	"regexp"

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

type StringFacet struct {
	Getter func(int) string
	Field  string
	Name   string
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
		cards = append(cards, Card{
			Name:   name,
			Value:  cnt,
			Filter: fmt.Sprintf("%s:%s", s.Field, q),
		})
	}
	slices.SortStableFunc(cards, func(a, b Card) bool {
		return a.Value > b.Value
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
