package app

import (
	"context"
	"fmt"
	"testing"
)

type x int

func (y x) String() string {
	return fmt.Sprint(int(y))
}

func TestLogging(t *testing.T) {
	ctx := context.Background()

	logger := Log.From(ctx)
	logger.Info().Msg("test")

	ctx2 := context.Background()
	Log.To(ctx2, logger)

	ctx = Log.WithInt(ctx, "a", 1)
	ctx = Log.WithStr(ctx, "b", "b")
	ctx = Log.WithStringer(ctx, "c", x(2))

	logger = Log.From(ctx)
	logger.Info().Msg("test 2")
}
