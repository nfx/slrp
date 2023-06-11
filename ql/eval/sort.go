package eval

import (
	"fmt"

	"github.com/nfx/slrp/ql/ast"
)

type Sorter[T any] struct {
	Asc         func(left, right T) bool
	Desc        func(left, right T) bool
	AscDefault  bool
	DescDefault bool
}

func (s Sorter[T]) Comparator(ob ast.OrderBy) func(left, right T) bool {
	if ob.Asc {
		return s.Asc
	}
	return s.Desc
}

type Sorters[T any] map[string]Sorter[T]

func (s Sorters[T]) Sort(sort ast.Sort) (func(left, right T) bool, error) {
	chain := []func(left, right T) bool{}
	for _, orderBy := range sort {
		o, ok := s[orderBy.Ident]
		if !ok {
			return nil, fmt.Errorf("no field: %s", orderBy.Ident)
		}
		chain = append(chain, o.Comparator(orderBy))
	}
	return func(left, right T) bool {
		for _, sorter := range chain {
			if sorter(left, right) {
				return true
			}
			if sorter(right, left) {
				break
			}
		}
		return false
	}, nil
}
