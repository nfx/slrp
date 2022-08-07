package eval

import (
	"fmt"

	"github.com/nfx/slrp/ql/internal"
	"golang.org/x/exp/slices"
)

type Dataset[T any] struct {
	Source    []T
	Accessors Accessors
	Facets    FacetRetrievers[T]
	Sorters   Sorters[T]
}

type QueryResult[T any] struct {
	Total   int
	Records []T
	Facets  []Facet
}

func (d Dataset[T]) Query(query string) (*QueryResult[T], error) {
	plan, err := internal.Parse(query)
	if err != nil {
		return nil, err
	}
	less, err := d.Sorters.Sort(plan.Sort)
	if err != nil {
		return nil, fmt.Errorf("sort: %w", err)
	}
	optimized := d.Transform(*plan)
	err, ok := d.IsFailure(optimized)
	if ok {
		return nil, err
	}
	result := []T{}
	for i := 0; i < len(d.Source); i++ {
		include, err := Filter(i, optimized)
		if err != nil {
			return nil, err
		}
		if !include {
			continue
		}
		result = append(result, d.Source[i])
	}
	// execute sort
	slices.SortStableFunc(result, less)
	topN := 10
	if plan.Limit == 0 {
		plan.Limit = 20
	}
	if plan.Limit >= len(result) {
		plan.Limit = len(result)
	}
	return &QueryResult[T]{
		Total:   len(result),
		Records: result[:plan.Limit],
		Facets:  d.Facets.Facets(result, topN),
	}, nil
}
