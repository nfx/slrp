package sorter

import (
	"reflect"
	"sort"
)

var withCaching = false

// rewrite this BS after Go1.18 in February
func Slice(s interface{}, fn func(int) Cmp) {
	if withCaching {
		rv := reflect.ValueOf(s)
		length := rv.Len()
		cache := make([]Cmp, length)
		for i := 0; i < length; i++ {
			cache[i] = fn(i)
		}
		sort.Slice(s, func(i, j int) bool {
			return cache[i].Less(cache[j])
		})
	} else {
		sort.Slice(s, func(i, j int) bool {
			return fn(i).Less(fn(j))
		})
	}
}

type Cmp interface {
	Less(o Cmp) bool
}

type FloatAsc float32

func (f FloatAsc) Less(o Cmp) bool {
	return float32(f) < float32(o.(FloatAsc))
}

type FloatDesc float32

func (f FloatDesc) Less(o Cmp) bool {
	return float32(f) > float32(o.(FloatDesc))
}

type IntAsc int

func (ia IntAsc) Less(o Cmp) bool {
	return int(ia) < int(o.(IntAsc))
}

type IntDesc int

func (ia IntDesc) Less(o Cmp) bool {
	return int(ia) > int(o.(IntDesc))
}

type StrAsc string

func (s StrAsc) Less(o Cmp) bool {
	return string(s) < string(o.(StrAsc))
}

type StrDesc string

func (s StrDesc) Less(o Cmp) bool {
	return string(s) < string(o.(StrDesc))
}

type Chain []Cmp

func (c Chain) Less(other Cmp) bool {
	o := other.(Chain)
	if len(c) == 0 || len(o) == 0 {
		return false
	}
	if c[0] == o[0] {
		return c[1:].Less(o[1:])
	}
	return c[0].Less(o[0])
}

type Chain2 []Cmp

func (c Chain2) Less(other Cmp) bool {
	o := other.(Chain2)
	for i := range c {
		if c[i].Less(o[i]) {
			return true
		}
		if o[i].Less(c[i]) {
			break
		}
	}
	return false
}
