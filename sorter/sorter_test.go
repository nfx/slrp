package sorter

import (
	"log"
	"testing"
)

type x struct {
	first, second, third int
}

func TestNoCaching(t *testing.T) {
	withCaching = false
	fixture := []x{
		{1, 6, 10},
		{1, 5, 9},
		{1, 5, 8},
		{2, 4, 7},
		{2, 3, 6},
		{2, 3, 5},
		{3, 2, 4},
		{3, 1, 3},
		{3, 1, 2},
	}
	Slice(fixture, func(i int) Cmp {
		return Chain2{
			IntDesc(fixture[i].first),
			IntDesc(fixture[i].second),
			IntAsc(fixture[i].third),
		}
	})
	log.Printf("%v", fixture)
}

func TestCaching(t *testing.T) {
	withCaching = true
	fixture := []x{
		{1, 6, 10},
		{1, 5, 9},
		{1, 5, 8},
		{2, 4, 7},
		{2, 3, 6},
		{2, 3, 5},
		{3, 2, 4},
		{3, 1, 3},
		{3, 1, 2},
	}
	Slice(fixture, func(i int) Cmp {
		return Chain2{
			IntDesc(fixture[i].first),
			IntDesc(fixture[i].second),
			IntAsc(fixture[i].third),
		}
	})
	log.Printf("%v", fixture)
	withCaching = false
}

func TestFloats(t *testing.T) {
	type y struct {
		a, b float32
	}
	fixture := []y{
		{0.345, 0.4},
		{0.098, 0.1},
		{0.098, 0.2},
		{0.255, 0.3},
	}
	Slice(fixture, func(i int) Cmp {
		return Chain{
			FloatAsc(fixture[i].a),
			FloatDesc(fixture[i].b),
		}
	})
	log.Printf("%v", fixture)
}

func TestStrs(t *testing.T) {
	type y struct {
		a, b string
	}
	fixture := []y{
		{"e", "q"},
		{"d", "w"},
		{"b", "a"},
		{"b", "b"},
	}
	Slice(fixture, func(i int) Cmp {
		return Chain{
			StrAsc(fixture[i].a),
			StrDesc(fixture[i].b),
		}
	})
	log.Printf("%v", fixture)
}
