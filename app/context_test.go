package app

import (
	"testing"
)

func TestMockCtx(t *testing.T) {
	ctx := MockCtx()

	a := newServiceA()
	ctx.Start(a)

	go ctx.Cancel()
	<-ctx.Done()
}

func TestMockStart(t *testing.T) {
	a := newServiceA()
	defer MockStart(a)()
}

func TestMockCtxNoSpin(t *testing.T) {
	ctx := MockCtx()
	defer ctx.Cancel()

	a := newServiceA()
	go a.Start(ctx)

	ctx.WaitAndSpin()
}
