package app

import (
	"context"
	"fmt"
	"reflect"
)

type Factories map[string]interface{}
type Singletons map[string]interface{}
type MockRuntime map[string]*mockCtx
type dependency struct {
	Rv  *reflect.Value
	Out reflect.Type
	In  []reflect.Type
}

func MockStartSpin[T any](this *T, other ...any) (*T, MockRuntime) {
	sgltns := Singletons{"this": this}
	for i, service := range other {
		sgltns[fmt.Sprintf("service%d", i+1)] = service
	}
	runtime := sgltns.MockStart()
	for k := range runtime {
		runtime[k].Spin()
	}
	return this, runtime
}

func (r MockRuntime) Context(main ...string) context.Context {
	if len(main) == 0 {
		main = append(main, "this")
	}
	this, ok := r[main[0]]
	if !ok {
		panic("no main service found")
	}
	return this.ctx
}

func (r MockRuntime) Stop() {
	for _, v := range r {
		v.Cancel()
	}
}

func (s Singletons) Monitor() *monitorServers {
	return &monitorServers{Singletons: s}
}

func (s Singletons) MockStart() MockRuntime {
	r := MockRuntime{}
	s["monitor"] = s.Monitor()
	for _, s := range s {
		c, ok := s.(configurable)
		if !ok {
			continue
		}
		err := c.Configure(nil)
		if err != nil {
			panic(err)
		}
	}
	for k, v := range s {
		service, ok := v.(Service)
		if !ok {
			continue
		}
		ctx := MockCtx()
		ctx.name = k
		service.Start(ctx)
		r[k] = ctx
	}
	return r
}

func (c Factories) Init() Singletons {
	deps, err := c.deps()
	if err != nil {
		panic(err)
	}
	inst := map[string]reflect.Value{}
	// resolve reflections on all dependencies
	for k := range deps {
		dep, err := c.resolve(k, deps, inst)
		if err != nil {
			panic(err)
		}
		inst[k] = dep
	}
	// de-reflect dependencies
	singletons := Singletons{}
	for k := range deps {
		singletons[k] = inst[k].Interface()
	}
	return singletons
}

func (c Factories) resolve(k string, deps map[string]dependency,
	inst map[string]reflect.Value) (reflect.Value, error) {
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
		for other_key, other_type := range deps {
			if other_type.Out != in {
				continue
			}
			dep, err := c.resolve(other_key, deps, inst)
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

func (c Factories) deps() (map[string]dependency, error) {
	types := map[string]dependency{}
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
		types[k] = d
	}
	return types, nil
}
