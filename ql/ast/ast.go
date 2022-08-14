package ast

import (
	"fmt"
	"strconv"
	"time"
)

type Cb func(Node) Node

type Node interface {
	Transform(cb Cb) Node
}

type Binary interface {
	LeftRight() (Node, Node)
}

type Query struct {
	Filter Node
	Sort   Sort
	Limit  int
}

func (n Query) Transform(cb Cb) Node {
	return cb(Query{
		Filter: cb(n.Filter.Transform(cb)),
		Sort:   n.Sort,
		Limit:  n.Limit,
	})
}

type Bool bool

const True = Bool(true)
const False = Bool(false)

func (n Bool) Transform(cb Cb) Node {
	return cb(n)
}

func (n Bool) And(o Bool) Node {
	return n && o
}

func (n Bool) String() string {
	return fmt.Sprintf("%t", n)
}

type Ident string

func (n Ident) Transform(cb Cb) Node {
	return cb(n)
}

func (n Ident) String() string {
	return string(n)
}

type String string

func (n String) Transform(cb Cb) Node {
	return cb(n)
}

func (n String) String() string {
	return fmt.Sprintf(`"%s"`, string(n))
}

type Number float64

func (n Number) Transform(cb Cb) Node {
	return cb(n)
}

func (n Number) String() string {
	return strconv.FormatFloat(float64(n), 'f', -1, 64)
}

type Duration time.Duration

func (n Duration) Transform(cb Cb) Node {
	return cb(n)
}

type And struct {
	Left, Right Node
}

func (n And) Transform(cb Cb) Node {
	return cb(And{cb(n.Left.Transform(cb)), cb(n.Right.Transform(cb))})
}

func (n And) String() string {
	return fmt.Sprintf("(%s AND %s)", n.Left, n.Right)
}

type Or struct {
	Left, Right Node
}

func (n Or) Transform(cb Cb) Node {
	return cb(Or{cb(n.Left.Transform(cb)), cb(n.Right.Transform(cb))})
}

func (n Or) String() string {
	return fmt.Sprintf("(%s OR %s)", n.Left, n.Right)
}

type Not struct {
	Left Node
}

func (n Not) Transform(cb Cb) Node {
	return cb(Not{cb(n.Left.Transform(cb))})
}

func (n Not) String() string {
	return fmt.Sprintf("NOT %s", n.Left)
}

type Equals struct {
	Left, Right Node
}

func (n Equals) Transform(cb Cb) Node {
	return cb(Equals{cb(n.Left.Transform(cb)), cb(n.Right.Transform(cb))})
}

func (n Equals) LeftRight() (Node, Node) {
	return n.Left, n.Right
}

func (n Equals) String() string {
	return fmt.Sprintf("%s = %s", n.Left, n.Right)
}

type Contains struct {
	Left, Right Node
}

func (n Contains) Transform(cb Cb) Node {
	return cb(Contains{cb(n.Left.Transform(cb)), cb(n.Right.Transform(cb))})
}

func (n Contains) LeftRight() (Node, Node) {
	return n.Left, n.Right
}

func (n Contains) String() string {
	return fmt.Sprintf("%s ~ %s", n.Left, n.Right)
}

type LessThan struct {
	Left, Right Node
}

func (n LessThan) Transform(cb Cb) Node {
	return cb(LessThan{cb(n.Left.Transform(cb)), cb(n.Right.Transform(cb))})
}

func (n LessThan) LeftRight() (Node, Node) {
	return n.Left, n.Right
}

func (n LessThan) String() string {
	return fmt.Sprintf("%s < %s", n.Left, n.Right)
}

type GreaterThan struct {
	Left, Right Node
}

func (n GreaterThan) Transform(cb Cb) Node {
	return cb(GreaterThan{cb(n.Left.Transform(cb)), cb(n.Right.Transform(cb))})
}

func (n GreaterThan) LeftRight() (Node, Node) {
	return n.Left, n.Right
}

func (n GreaterThan) String() string {
	return fmt.Sprintf("%s > %s", n.Left, n.Right)
}

type Sort []OrderBy

type OrderBy struct {
	Ident string
	Asc   bool
}
