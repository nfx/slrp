package app

import (
	"fmt"
	"reflect"
	"strings"
)

type Factories map[string]interface{}
type dependencies map[string]dependency
type instances map[string]reflect.Value
type dependency struct {
	Rv  *reflect.Value
	Out reflect.Type
	In  []reflect.Type
}

func (i instances) With(k string, v any) instances {
	i[k] = reflect.ValueOf(v)
	return i
}

func (i instances) Singletons() Singletons {
	// de-reflect dependencies
	singletons := Singletons{}
	for k := range i {
		singletons[k] = i[k].Interface()
	}
	return singletons
}

func (c Factories) Init() (Singletons, []string, error) {
	deps, err := c.dependencies()
	if err != nil {
		return nil, nil, err
	}
	order := deps.ordered()
	inst := instances{}
	// resolve reflections on all dependencies
	for k := range deps {
		dep, err := deps.resolve(k, inst)
		if err != nil {
			return nil, nil, err
		}
		inst[k] = dep
	}
	return inst.Singletons(), order, nil
}

func (c Factories) dependencies() (dependencies, error) {
	deps := dependencies{}
	for k, v := range c {
		rv := reflect.ValueOf(v)
		if rv.Kind() != reflect.Func {
			return nil, fmt.Errorf("%s is not a function", k)
		}
		t := rv.Type()
		if t.NumOut() > 2 {
			// two-output factories expect a second result as error
			return nil, fmt.Errorf("%s is not a factory", k)
		}
		d := dependency{Rv: &rv}
		d.Out = t.Out(0)
		for i := 0; i < t.NumIn(); i++ {
			d.In = append(d.In, t.In(i))
		}
		deps[k] = d
	}
	return deps, nil
}

func (d *dependency) matches(in reflect.Type) bool {
	isInIface := in.Kind() == reflect.Interface
	inImplsType := isInIface && d.Out.Implements(in)
	inEqualsType := d.Out == in
	return inImplsType || inEqualsType
}

// sort dependencies topologically according to Kahn's algorithm (1962)
// see https://doi.org/10.1145%2F368996.369025
func (deps dependencies) ordered() (order []string) {
	edges := map[string][]string{}
	indegree := map[string]int{}
	// materialize interface dependencies into neighbour adjacency graph
	for k := range deps {
		for _, in := range deps[k].In {
			for otherKey, otherType := range deps {
				if !otherType.matches(in) {
					continue
				}
				edges[k] = append(edges[k], otherKey)
				edges[otherKey] = append(edges[otherKey], k)
				indegree[k]++
			}
		}
	}
	q := []string{}
	// First we add items with no upstream dependencies
	for k := range deps {
		if indegree[k] == 0 {
			q = append(q, k)
		}
	}
	for len(q) > 0 {
		k := q[0]
		q = q[1:]
		order = append(order, k)
		for _, j := range edges[k] {
			// Reduce the indegree of adjacent nodes and add them to
			// the queue if their indegree becomes 0
			indegree[j]--
			if indegree[j] == 0 {
				q = append(q, j)
			}
		}
	}
	return order
}

func (deps dependencies) resolve(k string, inst instances) (reflect.Value, error) {
	ex, ok := inst[k]
	if ok {
		return ex, nil
	}
	t, ok := deps[k]
	if !ok {
		return reflect.Value{}, fmt.Errorf("%s is not declared", k)
	}
	args := []reflect.Value{}
	for _, in := range t.In {
		found := []string{}
		for otherKey, otherType := range deps {
			if !otherType.matches(in) {
				continue
			}
			dep, err := deps.resolve(otherKey, inst)
			if err != nil {
				return reflect.Value{}, fmt.Errorf(
					"cannot resolve %s because of %s: %s", k, otherKey, err)
			}
			found = append(found, otherKey)
			args = append(args, dep)
		}
		if len(found) > 1 {
			return reflect.Value{}, fmt.Errorf("multiple matches for %s: %s", in, strings.Join(found, ", "))
		}
		if len(found) == 0 {
			return reflect.Value{}, fmt.Errorf("cannot find %s for %s", in, k)
		}
	}
	var err error
	res := t.Rv.Call(args)
	inst[k] = res[0]
	if len(res) == 2 && res[1].Interface() != nil {
		err = res[1].Interface().(error)
	}
	return inst[k], err
}
