package eval

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

//go:generate go run ../generator/main.go Abc
type Abc struct {
	Bar    int
	Bore   int
	Zoom   string `facet:"Zooms"`
	Zuul   string `facet:"Zuuls"`
	Foo    string `facet:"Category"`
	Active bool
}

var fixture = AbcDataset{
	{1, 2, "abc", "xxx", "b", true},
	{2, 2, "bcd", "www", "b", true},
	{3, 4, "def", "xxx", "b", true},
	{4, 4, "feg", "zzz", "b", false},
	{5, 6, "egh", "zzz", "a", true},
	{7, 8, "aaa", "zzz", "a", true},
	{2, 1, "bbb", "yyy", "a", false},
}

func TestWrk(t *testing.T) {
	check{"Bar AND Active", nil, "incompatible branches: (Bar@number AND Active@bool)"}.run(t)
}

func TestItWorks(t *testing.T) {
	tests := []check{
		{"x:y", &QueryResult[Abc]{
			Records: []Abc{},
		}, ""},
		{"x $ y", nil, "syntax error: unexpected $unk: x <<<$>>> y"},
		{"Active ORDER BY b", nil, "sort: no field: b"},
		{"Bar:Zoom", nil, "incompatible equals: Bar@number = Zoom@string"},
		{"(Bar:Zoom) AND Active", nil, "incompatible equals: Bar@number = Zoom@string"},
		{"Active AND (Bar:Zoom)", nil, "incompatible equals: Bar@number = Zoom@string"},
		{"(Bar:Zoom) OR Active", nil, "incompatible equals: Bar@number = Zoom@string"},
		{"Active OR (Bar:Zoom)", nil, "incompatible equals: Bar@number = Zoom@string"},
		{"NOT (Bar:Zoom)", nil, "incompatible equals: Bar@number = Zoom@string"},
		{"Bar: Active", nil, "incompatible equals: Bar@number = Active@bool"},
		{"Bar ~ Active", nil, "incompatible contains: Bar@number ~ Active@bool"},
		{"Bar > Active", nil, "incompatible greater: Bar@number > Active@bool"},
		{"Bar < Active", nil, "incompatible less: Bar@number < Active@bool"},
		{"Bar AND Active", nil, "incompatible branches: (Bar@number AND Active@bool)"},
		{"Bar OR Zoom", nil, "incompatible branches: (Bar@number OR Zoom@string)"},
		{"NOT Bar", nil, "incompatible branches: NOT Bar@number"},
		{"w", &QueryResult[Abc]{ // special "match all" syntax
			Records: []Abc{
				{2, 2, "bcd", "www", "b", true},
			},
			Total: 1,
		}, ""},
		{"Zuul ~ w", &QueryResult[Abc]{
			Records: []Abc{
				{2, 2, "bcd", "www", "b", true},
			},
			Total: 1,
		}, ""},
		{"NOT Active AND Bar = 2", &QueryResult[Abc]{
			Records: []Abc{
				{2, 1, "bbb", "yyy", "a", false},
			},
			Total: 1,
		}, ""},
		{`"b" < "a"`, &QueryResult[Abc]{
			Records: []Abc{},
		}, ""},
		{`"a" > "b"`, &QueryResult[Abc]{
			Records: []Abc{},
		}, ""},
		{"b < a", &QueryResult[Abc]{
			Records: []Abc{},
		}, ""},
		{"a > b", &QueryResult[Abc]{
			Records: []Abc{},
		}, ""},
		{"Bar < Bore", &QueryResult[Abc]{
			Records: []Abc{
				{1, 2, "abc", "xxx", "b", true},
				{3, 4, "def", "xxx", "b", true},
				{5, 6, "egh", "zzz", "a", true},
				{7, 8, "aaa", "zzz", "a", true},
			},
			Facets: []Facet{
				{"Zuuls", []Card{
					{"zzz", 2, "Zuul:zzz"},
					{"xxx", 2, "Zuul:xxx"},
				}},
				{"Category", []Card{
					{"b", 2, "Foo:b"},
					{"a", 2, "Foo:a"},
				}},
			},
			Total: 4,
		}, ""},
		{"Bore>Bar", &QueryResult[Abc]{
			Records: []Abc{
				{1, 2, "abc", "xxx", "b", true},
				{3, 4, "def", "xxx", "b", true},
				{5, 6, "egh", "zzz", "a", true},
				{7, 8, "aaa", "zzz", "a", true},
			},
			Facets: []Facet{
				{"Zuuls", []Card{
					{"zzz", 2, "Zuul:zzz"},
					{"xxx", 2, "Zuul:xxx"},
				}},
				{"Category", []Card{
					{"b", 2, "Foo:b"},
					{"a", 2, "Foo:a"},
				}},
			},
			Total: 4,
		}, ""},
		{"Bar:Bore", &QueryResult[Abc]{
			Records: []Abc{
				{2, 2, "bcd", "www", "b", true},
				{4, 4, "feg", "zzz", "b", false},
			},
			Facets: []Facet{
				{"Category", []Card{{"b", 2, "Foo:b"}}},
			},
			Total: 2,
		}, ""},
		{"(Bar OR Bore) = 100", nil, "incompatible equals: incompatible branches: (Bar@number OR Bore@number) = 100"},
		{"Bar:Bore AND Active", &QueryResult[Abc]{
			Records: []Abc{
				{2, 2, "bcd", "www", "b", true},
			},
			Total: 1,
		}, ""},
		{"Bar:Bore OR Active", &QueryResult[Abc]{
			Records: []Abc{
				{1, 2, "abc", "xxx", "b", true},
				{2, 2, "bcd", "www", "b", true},
				{3, 4, "def", "xxx", "b", true},
				{4, 4, "feg", "zzz", "b", false},
				{5, 6, "egh", "zzz", "a", true},
				{7, 8, "aaa", "zzz", "a", true},
			},
			Total: 6,
			Facets: []Facet{
				{"Zuuls", []Card{
					{"zzz", 3, "Zuul:zzz"},
					{"xxx", 2, "Zuul:xxx"},
				}},
				{"Category", []Card{
					{"b", 4, "Foo:b"},
					{"a", 2, "Foo:a"},
				}},
			},
		}, ""},
		{"Bar:Bore ORDER BY Zoom DESC", &QueryResult[Abc]{
			Records: []Abc{
				{4, 4, "feg", "zzz", "b", false},
				{2, 2, "bcd", "www", "b", true},
			},
			Total: 2,
			Facets: []Facet{
				{"Category", []Card{
					{"b", 2, "Foo:b"},
				}},
			},
		}, ""},
		{"Zuul > yyy", &QueryResult[Abc]{
			Records: []Abc{
				{4, 4, "feg", "zzz", "b", false},
				{5, 6, "egh", "zzz", "a", true},
				{7, 8, "aaa", "zzz", "a", true},
			},
			Total: 3,
			Facets: []Facet{
				{"Zuuls", []Card{
					{"zzz", 3, "Zuul:zzz"},
				}},
				{"Category", []Card{
					{"a", 2, "Foo:a"},
				}},
			},
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.query, tt.run)
	}
}

type check struct {
	query string
	out   *QueryResult[Abc]
	err   string
}

func (tt check) run(t *testing.T) {
	got, err := fixture.Query(tt.query)
	if err != nil {
		assert.EqualError(t, err, tt.err)
		return
	}
	if !assert.Equal(t, tt.out, got) {
		x := []string{"Records: []Abc{"}
		for _, v := range got.Records {
			x = append(x,
				fmt.Sprintf(`{%d, %d, "%s", "%s", "%s", %t},`,
					v.Bar, v.Bore, v.Zoom, v.Zuul, v.Foo, v.Active))
		}
		x = append(x, "},")
		x = append(x, fmt.Sprintf("Total: %d,", got.Total))
		if len(got.Facets) > 0 {
			x = append(x, "Facets: []Facet{")
			for _, v := range got.Facets {
				c := []string{fmt.Sprintf(`{"%s", []Card{`, v.Name)}
				for _, card := range v.Top {
					c = append(c, fmt.Sprintf(`{"%s", %d, "%s"},`,
						card.Name, card.Value, card.Filter))
				}
				c = append(c, "}},")
				x = append(x, strings.Join(c, "\n"))
			}
			x = append(x, "},")
		}
		t.Logf("PLEASE FIX:\n%s", strings.Join(x, "\n"))
	}
}
