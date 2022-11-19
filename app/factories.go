package app

import (
	"fmt"
	"reflect"
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

func (c Factories) Init() Singletons {
	deps, err := c.dependencies()
	if err != nil {
		panic(err)
	}
	inst := instances{}
	// resolve reflections on all dependencies
	for k := range deps {
		dep, err := deps.resolve(k, inst)
		if err != nil {
			panic(err)
		}
		inst[k] = dep
	}
	return inst.Singletons()
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
		found := false
		isInIface := in.Kind() == reflect.Interface
		for other_key, other_type := range deps {
			inImplsType := isInIface && other_type.Out.Implements(in)
			inEqualsType := other_type.Out == in
			if !inImplsType && !inEqualsType {
				continue
			}
			dep, err := deps.resolve(other_key, inst)
			if err != nil {
				return reflect.Value{}, fmt.Errorf(
					"cannot resolve %s because of %s: %s", k, other_key, err)
			}
			args = append(args, dep)
			found = true
		}
		if !found {
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
