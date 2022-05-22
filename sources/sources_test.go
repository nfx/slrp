package sources

import (
	"context"
	"log"
	"net/http"
	"testing"

	"github.com/nfx/slrp/internal/qa"
)

func TestSpysme(t *testing.T) {
	qa.RunOnlyInDebug(t)
	ctx := context.Background()
	src := premproxy(ctx, &http.Client{})
	seen := map[string]int{}
	for x := range src.Generate(ctx) {
		y := x.String()
		seen[y] = seen[y] + 1
	}
	log.Printf("found: %d", len(seen))
}
