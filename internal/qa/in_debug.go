package qa

import (
	"os"
	"path"
	"testing"
)

func RunOnlyInDebug(t *testing.T) {
	ex, _ := os.Executable()
	if path.Base(ex) != "__debug_bin" {
		t.Skipf("%s is debug-only test", t.Name())
	}
}
