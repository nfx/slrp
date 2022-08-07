package eval

import (
	"github.com/nfx/slrp/ql/ast"
)

func (d Dataset[T]) Transform(src ast.Query) ast.Node {
	return src.Transform(func(raw ast.Node) ast.Node {
		switch n := raw.(type) {
		case ast.Query:
			matchAll, ok := d.StringFrom(n.Filter)
			if !ok {
				return n
			}
			return d.MatchAll(matchAll)
		case ast.And:
			if !d.IsBoolean(n.Left) || !d.IsBoolean(n.Right) {
				return invalidExpr("incompatible branches", n)
			}
			return n
		case ast.Or:
			if !d.IsBoolean(n.Left) || !d.IsBoolean(n.Right) {
				return invalidExpr("incompatible branches", n)
			}
			return n
		case ast.Not:
			if !d.IsBoolean(n.Left) {
				return invalidExpr("incompatible branches", n)
			}
			return n
		case ast.Equals:
			if d.IsString(n.Left) && d.IsString(n.Right) {
				return EqualString{n.Left, n.Right}
			}
			if d.IsNumber(n.Left) && d.IsNumber(n.Right) {
				return EqualNumber{n.Left, n.Right}
			}
			return invalidExpr("incompatible equals", n)
		case ast.LessThan:
			if d.IsString(n.Left) && d.IsString(n.Right) {
				return LessThanString{n.Left, n.Right}
			}
			if d.IsNumber(n.Left) && d.IsNumber(n.Right) {
				return LessThanNumber{n.Left, n.Right}
			}
			return invalidExpr("incompatible less", n)
		case ast.GreaterThan:
			if d.IsString(n.Left) && d.IsString(n.Right) {
				return GreaterThanString{n.Left, n.Right}
			}
			if d.IsNumber(n.Left) && d.IsNumber(n.Right) {
				return GreaterThanNumber{n.Left, n.Right}
			}
			return invalidExpr("incompatible greater", n)
		case ast.Contains:
			if !(d.IsString(n.Left) && d.IsString(n.Right)) {
				return invalidExpr("incompatible contains", n)
			}
			return n
		case ast.Ident:
			getter, ok := d.Accessors[n]
			if !ok {
				return ast.String(n)
			}
			return getter
		default:
			return raw
		}
	})
}

func (d Dataset[T]) IsFailure(n ast.Node) (error, bool) {
	var failures []Invalid
	n.Transform(func(n ast.Node) ast.Node {
		i, ok := n.(Invalid)
		if ok {
			failures = append(failures, i)
			return nil
		}
		return n
	})
	if failures == nil {
		return nil, false
	}
	return failures[0], true
}

// MatchAll runs optimisation for single-string-filters:
// if filter is just a text, search in all string fields
// EXAMPLE: text -> a~test OR b~text OR c~text
func (d Dataset[T]) MatchAll(m ast.Node) (or ast.Or) {
	for _, v := range d.Accessors {
		getter, ok := v.(StringGetter)
		if !ok {
			continue
		}
		cond := ast.Contains{
			Left:  getter,
			Right: m,
		}
		// there might be some other way
		if or.Left == nil {
			or.Left = cond
		} else if or.Right == nil {
			or.Right = cond
		} else {
			or = ast.Or{
				Left:  cond,
				Right: or,
			}
		}
	}
	return
}

func (d Dataset[T]) StringFrom(n ast.Node) (ast.Node, bool) {
	switch x := n.(type) {
	case ast.Ident:
		return ast.String(x), true
	case ast.String:
		return x, true
	default:
		return nil, false
	}
}

func (d Dataset[T]) IsString(raw ast.Node) bool {
	switch raw.(type) {
	case StringGetter, ast.String:
		return true
	default:
		return false
	}
}

func (d Dataset[T]) IsBoolean(raw ast.Node) bool {
	switch raw.(type) {
	case Invalid:
		return true // hack...
	case BooleanGetter, ast.Bool:
		return true
	case EqualNumber, EqualString, LessThanNumber,
		LessThanString, GreaterThanNumber, GreaterThanString:
		return true
	case ast.Equals, ast.LessThan, ast.GreaterThan,
		ast.Contains, ast.Not:
		return true
	default:
		return false
	}
}

func (d Dataset[T]) IsNumber(raw ast.Node) bool {
	switch raw.(type) {
	case NumberGetter, ast.Number:
		return true
	default:
		return false
	}
}
