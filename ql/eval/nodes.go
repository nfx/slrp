package eval

import (
	"fmt"

	"github.com/nfx/slrp/ql/ast"
)

type Accessor ast.Node // for now

type Accessors map[ast.Ident]Accessor

type BooleanGetter struct {
	Name string
	Func func(int) bool
}

func (n BooleanGetter) Transform(cb ast.Cb) ast.Node {
	return cb(n)
}

func (n BooleanGetter) String() string {
	return fmt.Sprintf("%s@bool", n.Name)
}

type NumberGetter struct {
	Name string
	Func func(int) float64
}

func (n NumberGetter) Transform(cb ast.Cb) ast.Node {
	return cb(n)
}

func (n NumberGetter) String() string {
	return fmt.Sprintf("%s@number", n.Name)
}

type StringGetter struct {
	Name string
	Func func(int) string
}

func (n StringGetter) Transform(cb ast.Cb) ast.Node {
	return cb(n)
}

func (n StringGetter) String() string {
	return fmt.Sprintf("%s@string", n.Name)
}

type EqualNumber struct {
	Left, Right ast.Node
}

func (n EqualNumber) Transform(cb ast.Cb) ast.Node {
	return cb(EqualNumber{
		cb(n.Left.Transform(cb)),
		cb(n.Right.Transform(cb)),
	})
}

type LessThanNumber struct {
	Left, Right ast.Node
}

func (n LessThanNumber) Transform(cb ast.Cb) ast.Node {
	return cb(LessThanNumber{
		cb(n.Left.Transform(cb)),
		cb(n.Right.Transform(cb)),
	})
}

type GreaterThanNumber struct {
	Left, Right ast.Node
}

func (n GreaterThanNumber) Transform(cb ast.Cb) ast.Node {
	return cb(GreaterThanNumber{
		cb(n.Left.Transform(cb)),
		cb(n.Right.Transform(cb)),
	})
}

type EqualString struct {
	Left, Right ast.Node
}

func (n EqualString) Transform(cb ast.Cb) ast.Node {
	return cb(EqualString{
		cb(n.Left.Transform(cb)),
		cb(n.Right.Transform(cb)),
	})
}

type LessThanString struct {
	Left, Right ast.Node
}

func (n LessThanString) Transform(cb ast.Cb) ast.Node {
	return cb(LessThanString{
		cb(n.Left.Transform(cb)),
		cb(n.Right.Transform(cb)),
	})
}

type GreaterThanString struct {
	Left, Right ast.Node
}

func (n GreaterThanString) Transform(cb ast.Cb) ast.Node {
	return cb(GreaterThanString{
		cb(n.Left.Transform(cb)),
		cb(n.Right.Transform(cb)),
	})
}

type Invalid struct {
	Message string
	Node    ast.Node
}

func (n Invalid) Transform(cb ast.Cb) ast.Node {
	return cb(n)
}

func invalidExpr(msg string, n ast.Node) Invalid {
	return Invalid{
		Message: fmt.Sprintf("%s: %s", msg, n),
		Node:    n,
	}
}

func (i Invalid) Error() string {
	return i.Message
}
