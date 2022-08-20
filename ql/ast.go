package ql

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pmux"
	"github.com/nfx/slrp/sorter"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// adapted mainly from SQL example from participle project (MIT)

type internalRow struct {
	record   reflect.Value
	fieldMap map[string]reflect.StructField
}

func (c *internalRow) HasField(name string) bool {
	_, ok := c.fieldMap[name]
	return ok
}

func (c *internalRow) Get(name string) interface{} {
	field := c.fieldMap[name]
	fieldValue := c.record.FieldByIndex(field.Index)
	if field.Type.Kind() == reflect.Struct {
		// TODO: test if filtering by proxy works
		return fmt.Sprintf("%s", fieldValue.Interface())
	}
	return fieldValue.Interface()
}

type Expression struct {
	And []AndCondition `@@ ( "AND" @@ )*`
}

func (e *Expression) eval(ctx internalRow) (bool, error) {
	for _, v := range e.And {
		result, err := v.eval(ctx)
		if err != nil {
			return false, err
		}
		if !result {
			return false, nil
		}
	}
	return true, nil
}

type Query struct {
	Expression *Expression `@@`
	// And     []*AndCondition `@@ ( "AND" @@ )*`
	OrderBy []OrderBy `("ORDER" "BY" @@ (Comma @@)*)?`
	Limit   int       `("LIMIT" @Int)?`
}

func (q *Query) applySort(
	fieldMap map[string]reflect.StructField,
	destination reflect.Value) error {
	if len(q.OrderBy) == 0 {
		return nil
	}
	var sortingRules []func(reflect.Value) sorter.Cmp
	for _, o := range q.OrderBy {
		cmp, err := o.cmp(fieldMap)
		if err != nil {
			return fmt.Errorf("cannot sort on %s: %w", o.Identifier, err)
		}
		sortingRules = append(sortingRules, cmp)
	}
	sorter.Slice(destination.Interface(), func(i int) sorter.Cmp {
		var chain sorter.Chain
		recordRV := destination.Index(i)
		for _, cmp := range sortingRules {
			chain = append(chain, cmp(recordRV))
		}
		return chain
	})
	return nil
}

type OrderBy struct {
	Identifier string `@Ident`
	Direction  string `@( "ASC" | "DESC" )?`
}

func Asc(f string) OrderBy {
	return OrderBy{f, "ASC"}
}

func Desc(f string) OrderBy {
	return OrderBy{f, "DESC"}
}

var comparators = map[reflect.Kind]map[string]func(interface{}) sorter.Cmp{
	reflect.Int: {
		"ASC": func(i interface{}) sorter.Cmp {
			return sorter.IntAsc(i.(int))
		},
		"DESC": func(i interface{}) sorter.Cmp {
			return sorter.IntDesc(i.(int))
		},
	},
	reflect.Int64: {
		"ASC": func(i interface{}) sorter.Cmp {
			return sorter.IntAsc(i.(int64))
		},
		"DESC": func(i interface{}) sorter.Cmp {
			return sorter.IntDesc(i.(int64))
		},
	},
	reflect.String: {
		"ASC": func(i interface{}) sorter.Cmp {
			return sorter.StrAsc(i.(string))
		},
		"DESC": func(i interface{}) sorter.Cmp {
			return sorter.StrDesc(i.(string))
		},
	},
}

func (o *OrderBy) cmp(fieldMap map[string]reflect.StructField) (func(reflect.Value) sorter.Cmp, error) {
	field, ok := fieldMap[o.Identifier]
	if !ok {
		return nil, fmt.Errorf("%s is not present in schema", o.Identifier)
	}
	if o.Direction == "" {
		o.Direction = "ASC"
	}
	index := field.Index
	kind := field.Type.Kind()
	switch field.Type.String() {
	case "int", "int64", "string":
		return func(record reflect.Value) sorter.Cmp {
			v := record.FieldByIndex(index)
			return comparators[kind][o.Direction](v.Interface())
		}, nil
	case "time.Duration":
		return func(record reflect.Value) sorter.Cmp {
			v := record.FieldByIndex(index)
			dur := v.Interface().(time.Duration)
			return comparators[reflect.Int64][o.Direction](int64(dur))
		}, nil
	case "time.Time":
		return func(record reflect.Value) sorter.Cmp {
			v := record.FieldByIndex(index)
			time := v.Interface().(time.Time)
			return comparators[reflect.Int64][o.Direction](time.Unix())
		}, nil
	case "pmux.Proxy":
		// TODO: it's hairy and must be fixed later.
		return func(record reflect.Value) sorter.Cmp {
			v := record.FieldByIndex(index)
			proxy := v.Interface().(pmux.Proxy)
			return comparators[reflect.Int64][o.Direction](int64(proxy))
		}, nil
	}
	return nil, fmt.Errorf("%s (%s) does not support sorting yet", field.Name, field.Type.Name())
}

type AndCondition struct {
	Or []Condition `@@ ( "OR" @@ )*`
}

func (ac *AndCondition) eval(ctx internalRow) (bool, error) {
	for _, v := range ac.Or {
		success, err := v.eval(ctx)
		if err != nil {
			return false, err
		}
		if success {
			return true, nil
		}
	}
	return false, nil
}

type Condition struct {
	Not     *Condition        `"NOT" @@`
	Operand *ConditionOperand `| @@`
}

func (c *Condition) eval(ctx internalRow) (bool, error) {
	if c.Not != nil {
		success, err := c.Not.eval(ctx)
		return !success, err
	}
	return c.Operand.eval(ctx)
}

type ConditionOperand struct {
	Left  *Value   `@@`
	Right *Compare `@@?`
}

func (co *ConditionOperand) eval(ctx internalRow) (bool, error) {
	left, err := co.Left.eval(ctx)
	if err != nil {
		return false, err
	}
	if co.Right == nil {
		success, ok := left.(bool)
		if ok {
			return success, nil
		}
		// when one supplies invalid operator -
		// it'll simply return no results for now
		// TODO: add position for syntax error
		return false, fmt.Errorf("left does not resolve to bool")
	}
	right, err := co.Right.Right.eval(ctx)
	if err != nil {
		return false, err
	}
	switch co.Right.Operator {
	case "<>", "!=":
		return left != right, nil
	case "=", ":":
		return left == right, nil
	case "~":
		return strings.Contains(fmt.Sprintf("%s", left),
			fmt.Sprintf("%s", right)), nil
	case ">":
		switch x := left.(type) {
		case float64:
			return x > right.(float64), nil
		case string:
			return x > right.(string), nil
		}
	case ">=":
		switch x := left.(type) {
		case float64:
			return x >= right.(float64), nil
		case string:
			return x >= right.(string), nil
		}
	case "<":
		switch x := left.(type) {
		case float64:
			return x < right.(float64), nil
		case string:
			return x < right.(string), nil
		}
	case "<=":
		switch x := left.(type) {
		case float64:
			return x <= right.(float64), nil
		case string:
			return x <= right.(string), nil
		}
	}
	return false, fmt.Errorf("cannot eval: %v %s %v", left, co.Right.Operator, right)
}

type Compare struct {
	Operator string `@Operator`
	Right    *Value `@@`
}

type Value struct {
	Number     *float64    `@(Float|Int)`
	Duration   *string     `| @Duration`
	String     *string     `| @String`
	Identifier *string     `| @Ident`
	Inner      *Expression `| "(" @@ ")"`
}

type stringer interface {
	String() string
}

func (v *Value) eval(ctx internalRow) (interface{}, error) {
	if v.Number != nil {
		return *v.Number, nil
	} else if v.Identifier != nil {
		if !ctx.HasField(*v.Identifier) {
			return *v.Identifier, nil
		}
		res := ctx.Get(*v.Identifier)
		if x, ok := res.(stringer); ok {
			return x.String(), nil
		}
		switch x := res.(type) {
		case int:
			return float64(x), nil
		case int64:
			return float64(x), nil
		case float64:
			return x, nil
		case time.Time:
			// compare epoch times
			return float64(x.Unix()), nil
		case time.Duration:
			// compare epoch times
			return float64(x), nil
		case string:
			return x, nil
		case bool:
			return x, nil
		}
		return nil, fmt.Errorf("%s is of unknown type: %+v",
			*v.Identifier, res)
	} else if v.String != nil {
		return *v.String, nil
	} else if v.Duration != nil {
		d, err := app.ParseDuration(*v.Duration)
		if err != nil {
			return nil, err
		}
		earlier := time.Now().Add(-1 * d).Unix()
		return float64(earlier), nil
	}
	if v.Inner == nil {
		return nil, fmt.Errorf("empty AST value")
	}
	return v.Inner.eval(ctx)
}

var parser *participle.Parser

func init() {
	parser = participle.MustBuild(&Query{},
		participle.Elide("Whitespace"),
		participle.UseLookahead(2),
		participle.Unquote("String"),
		participle.Lexer(lexer.MustSimple([]lexer.SimpleRule{
			{"Ident", `[a-zA-Z]\w*`},
			{"Duration", `(\d+[wdhms])+`},
			{"Int", `[-+]?\d+`},
			{"Float", `[-+]?(\d*\.)?\d+`},
			{"Number", `[-+]?(\d*\.)?\d+`},
			{"Operator", `>=|>|<=|<|~|:|=`},
			{"String", `\"(?:[^\"]|\\.)*\"`},
			{"Comma", `,`},
			{"Whitespace", `[ \t]+`},
		})),
	)
}

func parse(query string) (q Query, err error) {
	if query == "" {
		return Query{}, nil
	}
	err = parser.ParseString("", query, &q)
	return
}

type executeOption interface {
	apply(q *Query)
}

type DefaultOrder []OrderBy

func (o DefaultOrder) apply(q *Query) {
	if len(q.OrderBy) == 0 {
		q.OrderBy = o
	}
}

type DefaultLimit int

func (o DefaultLimit) apply(q *Query) {
	if q.Limit == 0 {
		q.Limit = int(o)
	}
}

func Execute[T any](src *[]T, dst *[]T, query string, facets func(*[]T),
	opts ...executeOption) (err error) {
	defer func() {
		p := recover()
		if perr, ok := p.(error); ok {
			err = perr
		}
	}()
	q, err := parse(query)
	if err != nil {
		return err
	}
	for _, o := range opts {
		o.apply(&q)
	}
	source := reflect.ValueOf(src).Elem()
	fieldMap := inferSchema(source)
	err = applyFilter(src, dst, &q, fieldMap)
	if err != nil {
		return err
	}
	dstRV := reflect.ValueOf(dst).Elem()
	err = q.applySort(fieldMap, dstRV)
	if err != nil {
		return err
	}
	if facets != nil {
		facets(dst)
	}
	applyLimit(dst, q.Limit)
	return nil
}

func inferSchema(source reflect.Value) map[string]reflect.StructField {
	typeOfSlice := source.Type()
	recordType := typeOfSlice.Elem()
	fieldMap := map[string]reflect.StructField{}
	for i := 0; i < recordType.NumField(); i++ {
		field := recordType.Field(i)
		fieldMap[field.Name] = field
	}
	return fieldMap
}

func applyFilter[T any](src *[]T, dst *[]T, q *Query,
	fieldMap map[string]reflect.StructField) error {
	if q.Expression == nil {
		// empty expression is matching all
		q.Expression = &Expression{}
	}
	for i := 0; i < len(*src); i++ {
		record := (*src)[i]
		recordRV := reflect.ValueOf(record)
		success, err := q.Expression.eval(internalRow{recordRV, fieldMap})
		if err != nil {
			return fmt.Errorf("error filtering %d record: %w", i, err)
		}
		if success {
			*dst = append(*dst, record)
		}
	}
	return nil
}

func applyLimit[T any](dst *[]T, limit int) {
	if limit == 0 {
		// by default limit should be something small, like 100 records
		limit = 100
	}
	if len(*dst) < limit {
		// and be adjusted to available data
		limit = len(*dst)
	}
	*dst = (*dst)[0:limit]
}
