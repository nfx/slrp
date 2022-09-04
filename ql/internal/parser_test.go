package internal

import (
	"testing"
	"time"

	. "github.com/nfx/slrp/ql/ast"
	"github.com/stretchr/testify/assert"
)

//go:generate goyacc -o parser.go parser.y

func TestWrk(t *testing.T) {
	// yyDebug = 3
	query, err := Parse(`a LIMIT 1`)
	if assert.NoError(t, err) {
		t.Logf("X: %#v\n", query)
		assert.Equal(t, &Query{
			Filter: Ident("a"),
			Limit:  1,
		}, query)
	}
}

func TestParsing(t *testing.T) {
	tests := []struct {
		in  string
		out *Query
		err string
	}{
		{"!", nil, "syntax error: unexpected $end: !<<<>>>"},
		{"@", nil, "syntax error: unexpected $unk: <<<@>>>"},
		{"10", &Query{Filter: Number(10)}, ""},
		{"10.1", &Query{Filter: Number(10.1)}, ""},
		{"10(^)1", nil, "syntax error: unexpected '(': 10<<<(>>>^)1"},
		{"name", &Query{Filter: Ident("name")}, ""},
		{"na me", nil, "syntax error: unexpected IDENT: na <<<me>>>"},
		{"n0", nil, "syntax error: unexpected NUMBER: n<<<0>>>"},
		{`"name"`, &Query{Filter: String("name")}, ""},
		{"'n0'", nil, "syntax error: unexpected $end: <<<'n0'>>>"},
		{"`n0`", nil, "syntax error: unexpected $end: <<<`n0`>>>"},
		{`(()())`, nil, "syntax error: unexpected ')': ((<<<)>>>())"},
		{`((a)())`, nil, "syntax error: unexpected '(': ((a)<<<(>>>))"},
		{"(1)", &Query{Filter: Number(1)}, ""},
		{"!(2)", &Query{Filter: Not{Left: Number(2)}}, ""},
		{"!w", &Query{Filter: Not{Left: Ident("w")}}, ""},
		{"NOT w", &Query{Filter: Not{Left: Ident("w")}}, ""},
		{"5s", &Query{Filter: Duration(5 * time.Second)}, ""},
		{"5h", &Query{Filter: Duration(5 * time.Hour)}, ""},
		{"5d", &Query{Filter: Duration(5 * 24 * time.Hour)}, ""},
		{"5w", &Query{Filter: Duration(5 * 7 * 24 * time.Hour)}, ""},
		{"5x", nil, "syntax error: unexpected IDENT: 5<<<x>>>"},
		{"LIMIT 7", nil, "syntax error: unexpected LIMIT: <<<LIMIT>>> 7"},
		{"a LIMIT", nil, "syntax error: unexpected $end, expecting NUMBER: ..LIMIT<<<>>>"},
		{"a LIMIT b", nil, "syntax error: unexpected IDENT, expecting NUMBER: ..IMIT <<<b>>>"},
		{"a LIMIT 7", &Query{
			Filter: Ident("a"),
			Limit:  7,
		}, ""},
		{"a ORDER BY", nil, "syntax error: unexpected $end, expecting IDENT: ..ER BY<<<>>>"},
		{"a ORDER BY 1", nil, "syntax error: unexpected NUMBER, expecting IDENT: ..R BY <<<1>>>"},
		{"a ORDER BY a X", nil, "syntax error: unexpected IDENT: ..BY a <<<X>>>"},
		{"a ORDER BY DESC", nil, "syntax error: unexpected DESC, expecting IDENT: ..R BY <<<DESC>>>"},
		{"a ORDER BY a DESC, b", &Query{
			Filter: Ident("a"),
			Sort: Sort{
				OrderBy{Ident: "a", Asc: false},
				OrderBy{Ident: "b", Asc: true},
			},
		}, ""},
		{"Bar > 2 AND !Active", &Query{
			Filter: And{
				Left: GreaterThan{
					Left:  Ident("Bar"),
					Right: Number(2),
				},
				Right: Not{Left: Ident("Active")},
			},
		}, ""},
		{"AND", nil, "syntax error: unexpected AND: <<<AND>>>"},
		{"a AND", nil, "syntax error: unexpected $end: a AND<<<>>>"},
		{"a AND OR", nil, "syntax error: unexpected OR: .. AND <<<OR>>>"},
		{"1 AND 2", &Query{
			Filter: And{
				Left:  Number(1),
				Right: Number(2),
			},
		}, ""},
		{"1 AND !2", &Query{Filter: And{
			Left: Number(1),
			Right: Not{
				Left: Number(2),
			},
		},
		}, ""},
		{"NOT (1 AND !2)", &Query{
			Filter: Not{
				Left: And{
					Left: Number(1),
					Right: Not{
						Left: Number(2),
					},
				},
			},
		}, ""},
		{"1 OR 2", &Query{
			Filter: Or{
				Left:  Number(1),
				Right: Number(2),
			},
		}, ""},
		{"a ~ b", &Query{
			Filter: Contains{
				Left:  Ident("a"),
				Right: Ident("b"),
			},
		}, ""},
		{"a OR b AND c OR d", &Query{
			Filter: Or{
				Left: Or{
					Left: Ident("a"),
					Right: And{
						Left:  Ident("b"),
						Right: Ident("c"),
					},
				},
				Right: Ident("d"),
			},
		}, ""},
		{"a OR NOT b AND c", &Query{
			Filter: Or{
				Left: Ident("a"),
				Right: And{
					Left:  Not{Left: Ident("b")},
					Right: Ident("c"),
				},
			},
		}, ""},
		{"a:1 OR b=2", &Query{
			Filter: Or{
				Left: Equals{
					Left:  Ident("a"),
					Right: Number(1),
				},
				Right: Equals{
					Left:  Ident("b"),
					Right: Number(2),
				},
			},
		}, ""},
		{"a <> 1 OR c < 3 OR d > 4", &Query{
			Filter: Or{
				Left: Or{
					Left: Not{
						Left: Equals{
							Left:  Ident("a"),
							Right: Number(1),
						},
					},
					Right: LessThan{
						Left:  Ident("c"),
						Right: Number(3),
					},
				},
				Right: GreaterThan{
					Left:  Ident("d"),
					Right: Number(4),
				},
			},
		}, ""},
		{`a != "b" AND !c`, &Query{
			Filter: And{
				Left: Not{
					Left: Equals{
						Left:  Ident("a"),
						Right: String("b"),
					},
				},
				Right: Not{
					Left: Ident("c"),
				},
			},
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := Parse(tt.in)
			if err != nil {
				assert.EqualError(t, err, tt.err)
				return
			}
			assert.Equal(t, tt.out, got)
		})
	}
}

func TestYyErrorMessageNotVerbose(t *testing.T) {
	prev := yyErrorVerbose
	yyErrorVerbose = false
	assert.Equal(t, "syntax error", yyErrorMessage(0, 0))
	yyErrorVerbose = prev
}

func TestYyErrorMessageCoverage_001(t *testing.T) {
	prev := yyErrorVerbose
	yyErrorVerbose = true

	state := 1
	yyPactPrev := yyPact[state]
	yyDefPrev := yyDef[state]
	yyExcaPrev := yyExca
	yyExca = [...]int8{
		0, -100,
		-1, int8(state),
		-1, 0,
	}
	yyDef[state] = -2
	yyPact[state] = -100
	assert.Equal(t, "syntax error: unexpected tok-0", yyErrorMessage(state, 0))
	yyErrorVerbose = prev
	yyDef[state] = yyDefPrev
	yyPact[state] = yyPactPrev
	yyExca = yyExcaPrev
}

func TestYyErrorMessageCoverage_002(t *testing.T) {
	prev := yyErrorVerbose
	yyErrorVerbose = true

	state := 1
	yyPactPrev := yyPact[state]
	yyDefPrev := yyDef[state]
	yyExcaPrev := yyExca
	yyExca = [...]int8{
		0, -100,
		-1, int8(state),
		-1, 0,
	}
	yyDef[state] = -2
	yyPact[state] = -100
	assert.Equal(t, "syntax error: unexpected tok-0", yyErrorMessage(state, 0))
	yyErrorVerbose = prev
	yyDef[state] = yyDefPrev
	yyPact[state] = yyPactPrev
	yyExca = yyExcaPrev
}
