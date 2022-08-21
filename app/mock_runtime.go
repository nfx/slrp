package app

import (
	"context"
	"fmt"
)

type MockRuntime map[string]*mockCtx

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
