package checker

import (
	"context"
	"testing"

	"github.com/nfx/slrp/pmux"
)

func TestFailure(t *testing.T) {
	var proxy pmux.Proxy
	defer pmux.SetupHttpProxy(&proxy)()
	c := NewChecker()

	ctx := context.Background()
	_, err := c.Check(ctx, proxy)
	if err == nil {
		t.Fatal("expected error")
	}
}
