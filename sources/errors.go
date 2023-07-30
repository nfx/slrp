package sources

import (
	"errors"
	"fmt"

	"github.com/nfx/slrp/pmux"
	"github.com/rs/zerolog"
)

type errorContext interface {
	Apply(e *zerolog.Event)
}

type intEC struct {
	key   string
	value int
}

func (a intEC) Apply(e *zerolog.Event) {
	e.Int(a.key, a.value)
}

type strEC struct {
	key   string
	value string
}

func (a strEC) Apply(e *zerolog.Event) {
	e.Str(a.key, a.value)
}

type sourceError struct {
	msg    string
	fields []errorContext
	skip   bool
}

func (se sourceError) Proxy() pmux.Proxy {
	for _, v := range se.fields {
		switch x := v.(type) {
		case strEC:
			return pmux.NewProxyFromURL(x.value)
		default:
			continue
		}
	}
	return 0
}

func (se sourceError) Error() string {
	ctx := se.msg
	for _, v := range se.fields {
		switch x := v.(type) {
		case intEC:
			ctx = fmt.Sprintf("%s %s=%d", ctx, x.key, x.value)
		case strEC:
			ctx = fmt.Sprintf("%s %s=%s", ctx, x.key, x.value)
		}
	}
	if se.skip {
		ctx = fmt.Sprintf("%s (skip)", ctx)
	}
	return ctx
}

func newErr(msg string, ctx ...errorContext) sourceError {
	return sourceError{
		msg:    msg,
		fields: ctx,
	}
}

func wrapError(err error, ctx ...errorContext) sourceError {
	switch x := err.(type) {
	case sourceError:
		x.fields = append(x.fields, ctx...)
		return x
	default:
		return newErr(err.Error(), ctx...)
	}
}

func skipErr(err error, ctx ...errorContext) sourceError {
	se := wrapError(err, ctx...)
	se.skip = true
	return se
}

func skipError(msg string, ctx ...errorContext) sourceError {
	return skipErr(errors.New(msg), ctx...)
}
