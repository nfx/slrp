package ql

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/sorter"
	"github.com/stretchr/testify/assert"
)

type x struct {
	First, Second, Third int
	Fifth                string
}

var fixture = []x{
	{1, 6, 10, "aabb"},
	{1, 5, 9, "ccdd"},
	{1, 5, 8, "eeff"},
	{2, 4, 7, "gghh"},
	{2, 3, 6, "iijj"},
	{2, 3, 5, "hhii"},
	{3, 2, 4, "ffgg"},
	{3, 1, 3, "ddee"},
	{3, 1, 2, "bbcc"},
}

type tt struct {
	query string
	err   error
}

func TestParse(t *testing.T) {
	tests := []tt{
		// time.ParseDuration("5h30m40s")
		// {"Second > 22", nil, nil},
		// {"Second > 22 ORDER BY First, Second DESC LIMIT 2", nil, nil},
		{"Fifth~a OR Fifth~d ORDER BY First, Second DESC LIMIT 2", nil},
		//{"foo:bar AND NOT (bar=\"baz\" OR foo ~ 1) ORDER BY foo, bar DESC", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			var result []x
			err := Execute(&fixture, &result, tt.query, func(t *[]x) {})
			if err != tt.err {
				t.Errorf("Parse() error = %v", err)
				return
			}
		})
	}
}

func newInternalRow(record any) internalRow {
	rv := reflect.ValueOf(record)
	fieldMap := map[string]reflect.StructField{}
	recordType := rv.Type()
	for i := 0; i < recordType.NumField(); i++ {
		field := recordType.Field(i)
		fieldMap[field.Name] = field
	}
	return internalRow{rv, fieldMap}
}

func TestInternalRowGet(t *testing.T) {
	type foo struct {
		A string
		B time.Time
	}
	x := foo{}
	x.A = "abc"
	x.B = time.Time{}
	ir := newInternalRow(x)

	res := ir.Get("B")

	assert.Equal(t, "0001-01-01 00:00:00 +0000 UTC", res)
}

func ref(str string) *string {
	return &str
}

func TestExpressionEvalError(t *testing.T) {
	e := Expression{
		And: []AndCondition{
			{
				Or: []Condition{
					{
						Not: &Condition{
							Operand: &ConditionOperand{
								Left: &Value{
									Duration: ref("abc"),
								},
							},
						},
					},
				},
			},
		},
	}

	res, err := e.eval(newInternalRow(x{1, 2, 3, "d"}))
	assert.EqualError(t, err, "left does not resolve to bool")
	assert.Equal(t, false, res)
}

func TestQueryApplySort(t *testing.T) {
	ir := newInternalRow(x{1, 2, 3, "d"})
	err := (&Query{}).applySort(ir.fieldMap, ir.record)
	assert.NoError(t, err)
}

func TestQueryApplySortEasyFail(t *testing.T) {
	ir := newInternalRow(x{1, 2, 3, "d"})
	err := (&Query{
		OrderBy: []OrderBy{
			{"empty", "invalid"},
		},
	}).applySort(ir.fieldMap, ir.record)
	assert.EqualError(t, err, "cannot sort on empty: empty is not present in schema")
}

func TestOrderByIntAsc(t *testing.T) {
	ir := newInternalRow(x{1, 2, 3, "d"})
	firstAsc := Asc("First")
	cmp, err := firstAsc.cmp(ir.fieldMap)
	assert.NoError(t, err)
	less := cmp(ir.record).Less(sorter.IntAsc(100))
	assert.Equal(t, true, less)
}

func TestOrderByStringAsc(t *testing.T) {
	ir := newInternalRow(x{1, 2, 3, "d"})
	fifthAsc := Asc("Fifth")
	cmp, err := fifthAsc.cmp(ir.fieldMap)
	assert.NoError(t, err)
	less := cmp(ir.record).Less(sorter.StrAsc("e"))
	assert.Equal(t, true, less)
}

func TestOrderByStringDesc(t *testing.T) {
	ir := newInternalRow(x{1, 2, 3, "d"})
	fifthDesc := Desc("Fifth")
	cmp, err := fifthDesc.cmp(ir.fieldMap)
	assert.NoError(t, err)
	less := cmp(ir.record).Less(sorter.StrDesc("e")) // TODO: bug o_O
	assert.Equal(t, true, less)
}

type y struct {
	Dur   time.Duration
	Seen  time.Time
	Proxy pmux.Proxy
}

func TestOrderByDurationAsc(t *testing.T) {
	ir := newInternalRow(y{1 * time.Second, time.Now(), pmux.HttpProxy("127.0.0.1:1234")})
	durAsc := Asc("Dur")
	cmp, err := durAsc.cmp(ir.fieldMap)
	assert.NoError(t, err)
	less := cmp(ir.record).Less(sorter.IntAsc(100 * time.Second))
	assert.Equal(t, true, less)
}

func TestOrderByTimeDescComparesAsUnix(t *testing.T) {
	n := time.Now()
	ir := newInternalRow(y{1 * time.Second, n, pmux.HttpProxy("127.0.0.1:1234")})
	seenDesc := Desc("Seen")
	cmp, err := seenDesc.cmp(ir.fieldMap)
	assert.NoError(t, err)
	less := cmp(ir.record).Less(sorter.IntDesc(n.Unix() - 100))
	assert.Equal(t, true, less)
}

func TestOrderByProxyAscIsComparedAsInt64(t *testing.T) {
	n := time.Now()
	ir := newInternalRow(y{1 * time.Second, n, pmux.HttpProxy("127.0.0.1:1234")})
	proxyAsc := Asc("Proxy")
	cmp, err := proxyAsc.cmp(ir.fieldMap)
	assert.NoError(t, err)
	less := cmp(ir.record).Less(sorter.IntAsc(pmux.HttpProxy("127.0.0.2:1234")))
	assert.Equal(t, true, less)
}

func TestOrderByProxyDescIsComparedAsInt64(t *testing.T) {
	n := time.Now()
	ir := newInternalRow(y{1 * time.Second, n, pmux.HttpProxy("127.0.0.1:1234")})
	proxyDesc := Desc("Proxy")
	cmp, err := proxyDesc.cmp(ir.fieldMap)
	assert.NoError(t, err)
	less := cmp(ir.record).Less(sorter.IntDesc(pmux.HttpProxy("127.0.0.0:1234")))
	assert.Equal(t, true, less)
}

func TestOrderByNotSupported(t *testing.T) {
	type w struct {
		A chan string
	}
	ir := newInternalRow(w{make(chan string)})
	proxyDesc := Desc("A")
	_, err := proxyDesc.cmp(ir.fieldMap)
	assert.EqualError(t, err, "A () does not support sorting yet")
}

func TestConditionOperandEvalLeftErr(t *testing.T) {
	ir := newInternalRow(x{1, 2, 3, "d"})
	co := &ConditionOperand{
		Left: &Value{},
	}
	_, err := co.eval(ir)
	assert.EqualError(t, err, "empty AST value")
}

func TestConditionOperandEvalLeftIsBoolRightIsNull(t *testing.T) {
	type w struct {
		A bool
	}
	ir := newInternalRow(w{true})
	co := &ConditionOperand{
		Left: &Value{
			Identifier: ref("A"),
		},
	}
	res, err := co.eval(ir)
	assert.NoError(t, err)
	assert.Equal(t, true, res)
}

func TestConditionOperandEvalLeftIsStringRightIsNull(t *testing.T) {
	type w struct {
		A bool
	}
	ir := newInternalRow(w{true})
	co := &ConditionOperand{
		Left: &Value{
			Identifier: ref("B"),
		},
	}
	_, err := co.eval(ir)
	assert.EqualError(t, err, "left does not resolve to bool")
}

// given very limited resources, we either evaluate strings or float64
func TestConditionOperandEvalStrings(t *testing.T) {
	ir := newInternalRow(x{1, 2, 3, "d"})
	type tc struct {
		left     string
		operator string
		right    string
	}
	for _, tt := range []tc{
		{"a", "<>", "b"},
		{"a", "!=", "b"},
		{"a", ":", "a"},
		{"a", "=", "a"},
		{"abc", "~", "a"},
		{"b", ">", "a"},
		{"b", ">=", "a"},
		{"a", "<", "b"},
		{"a", "<=", "b"},
	} {
		caseName := fmt.Sprintf("%s %s %s", tt.left, tt.operator, tt.right)
		t.Run(caseName, func(t *testing.T) {
			co := &ConditionOperand{
				Left: &Value{
					String: &tt.left,
				},
				Right: &Compare{
					Operator: tt.operator,
					Right: &Value{
						String: &tt.right,
					},
				},
			}
			result, err := co.eval(ir)
			assert.NoError(t, err, caseName)
			assert.True(t, result)
		})
	}
}

// given very limited resources, we either evaluate strings or float64
func TestConditionOperandEvalFloat64(t *testing.T) {
	ir := newInternalRow(x{1, 2, 3, "d"})
	type tc struct {
		left     float64
		operator string
		right    float64
	}
	for _, tt := range []tc{
		{1, "<>", 2},
		{1, "!=", 2},
		{1, ":", 1},
		{1, "=", 1},
		{2, ">", 1},
		{2, ">=", 1},
		{1, "<", 2},
		{1, "<=", 2},
	} {
		caseName := fmt.Sprintf("%f %s %f", tt.left, tt.operator, tt.right)
		t.Run(caseName, func(t *testing.T) {
			co := &ConditionOperand{
				Left: &Value{
					Number: &tt.left,
				},
				Right: &Compare{
					Operator: tt.operator,
					Right: &Value{
						Number: &tt.right,
					},
				},
			}
			result, err := co.eval(ir)
			assert.NoError(t, err, caseName)
			assert.True(t, result)
		})
	}
}

func TestConditionOperandEvalWrongOperator(t *testing.T) {
	type w struct {
		A bool
	}
	ir := newInternalRow(w{true})
	co := &ConditionOperand{
		Left: &Value{
			Identifier: ref("B"),
		},
		Right: &Compare{
			Operator: "kiss",
			Right: &Value{
				Identifier: ref("C"),
			},
		},
	}
	_, err := co.eval(ir)
	assert.EqualError(t, err, "cannot eval: B kiss C")
}
