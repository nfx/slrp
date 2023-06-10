package eval

import (
	"fmt"
	"strings"

	"github.com/nfx/slrp/ql/ast"
)

func Filter(record int, filter ast.Node) (bool, error) {
	t := filter.Transform(func(e ast.Node) ast.Node {
		switch op := e.(type) {
		case ast.Bool:
			return op
		case ast.String:
			return op
		case ast.Number:
			return op
		case StringGetter:
			return ast.String(op.Func(record))
		case NumberGetter:
			return ast.Number(op.Func(record))
		case BooleanGetter:
			return ast.Bool(op.Func(record))
		case ast.Contains:
			left := op.Left.(ast.String)
			right := op.Right.(ast.String)
			leftLower := strings.ToLower(string(left))
			rightLower := strings.ToLower(string(right))
			return ast.Bool(strings.Contains(leftLower, rightLower))
		case EqualString:
			left := op.Left.(ast.String)
			right := op.Right.(ast.String)
			return ast.Bool(left == right)
		case LessThanString:
			left := op.Left.(ast.String)
			right := op.Right.(ast.String)
			return ast.Bool(left < right)
		case GreaterThanString:
			left := op.Left.(ast.String)
			right := op.Right.(ast.String)
			return ast.Bool(left > right)
		case EqualNumber:
			left := op.Left.(ast.Number)
			right := op.Right.(ast.Number)
			return ast.Bool(left == right)
		case LessThanNumber:
			left := op.Left.(ast.Number)
			right := op.Right.(ast.Number)
			return ast.Bool(left < right)
		case GreaterThanNumber:
			left := op.Left.(ast.Number)
			right := op.Right.(ast.Number)
			return ast.Bool(left > right)
		case ast.Not:
			b := op.Left.(ast.Bool)
			return ast.Bool(!b)
		case ast.And:
			left := op.Left.(ast.Bool)
			right := op.Right.(ast.Bool)
			return ast.Bool(left && right)
		case ast.Or:
			left := op.Left.(ast.Bool)
			right := op.Right.(ast.Bool)
			return ast.Bool(left || right)
		case Invalid:
			return op
		case ast.Query:
			return op.Filter
		default:
			return ast.False
		}
	})
	switch op := t.(type) {
	case ast.Bool:
		return bool(op), nil
	default:
		return false, fmt.Errorf("unknown return: %T", t)
	}
}
