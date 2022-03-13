package app

import "context"
import "github.com/rs/zerolog/log"

type Service interface {
	Start(Context)
}

type Context interface {
	Ctx() context.Context

	// Propagates Done channel from parent context.
	Done() <-chan struct{}

	// Heartbeat used as signal to Fabric that marks related service for the next storage
	// flush event, in case BinaryMarshaller/BinaryUnmarshaller interfaces are implemented.
	// Second major use is as unit testing blocking hook.
	Heartbeat()
	// TODO: method for getting last updated, so UI HTTP pooling is relaxed
}

func MockCtx() *mockCtx {
	ctx, cancel := context.WithCancel(context.Background())
	return &mockCtx{
		ctx:    ctx,
		Cancel: cancel,
		Wait:   make(chan bool),
	}
}

func MockStart(s Service) func() {
	ctx := MockCtx()
	ctx.Start(s)
	return func() {
		ctx.Cancel()
	}
}

type mockCtx struct {
	ctx    context.Context
	Cancel func()
	Wait   chan bool
	name   string
	spin   bool
}

func (a *mockCtx) Start(s Service) {
	s.Start(a)
	a.Spin()
}

func (a *mockCtx) Spin() {
	a.spin = true
}

func (a *mockCtx) Ctx() context.Context {
	return a.ctx
}
func (a *mockCtx) Done() <-chan struct{} {
	return a.ctx.Done()
}
func (a *mockCtx) Heartbeat() {
	if a.spin {
		return
	}
	log.Trace().Str("service", a.name).Msg("heartbeat mock")
	a.Wait <- true
}

type serviceContext struct {
	ctx  context.Context
	name string
	sync chan string
}

func (sc *serviceContext) Ctx() context.Context {
	return sc.ctx
}

func (sc *serviceContext) Done() <-chan struct{} {
	return sc.ctx.Done()
}

func (sc *serviceContext) Heartbeat() {
	sc.sync <- sc.name
}
