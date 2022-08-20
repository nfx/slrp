package main

import (
	"os"
	"testing"

	"github.com/nfx/slrp/internal/qa"
)

func TestMain(t *testing.T) {
	qa.RunOnlyInDebug(t)
	os.Setenv("SLRP_PPROF_ENABLE", "true")
	os.Setenv("SLRP_HISTORY_LIMIT", "10000")
	os.Setenv("SLRP_LOG_LEVEL", "debug")
	os.Setenv("SLRP_LOG_FORMAT", "file")        // TODO: eek, make it better
	os.Setenv("SLRP_LOG_FILE", "/tmp/$APP.log") // TODO: eek, make it better

	main()
}
