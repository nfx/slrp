package app

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var Log passLogger

type passLogger struct{}

func (l passLogger) From(ctx context.Context) zerolog.Logger {
	logger, ok := ctx.Value(l).(zerolog.Logger)
	if !ok {
		return log.Logger
	}
	return logger
}

func (l passLogger) To(ctx context.Context, log zerolog.Logger) context.Context {
	return context.WithValue(ctx, l, log)
}

func (l passLogger) Nest(ctx context.Context, cb func(zc zerolog.Context) zerolog.Context) context.Context {
	log := l.From(ctx)
	zc := cb(log.With())
	return l.To(ctx, zc.Logger())
}

func (l passLogger) WithStringer(ctx context.Context, key string, value fmt.Stringer) context.Context {
	return l.Nest(ctx, func(zc zerolog.Context) zerolog.Context {
		return zc.Stringer(key, value)
	})
}

func (l passLogger) WithStr(ctx context.Context, key, value string) context.Context {
	return l.Nest(ctx, func(zc zerolog.Context) zerolog.Context {
		return zc.Str(key, value)
	})
}

func (l passLogger) WithInt(ctx context.Context, key string, value int) context.Context {
	return l.Nest(ctx, func(zc zerolog.Context) zerolog.Context {
		return zc.Int(key, value)
	})
}
