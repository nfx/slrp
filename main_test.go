package main

import (
	"os"
	"testing"
)

func TestMain(t *testing.T) {
	os.Setenv("SLRP_HISTORY_LIMIT", "10000")
	os.Setenv("SLRP_LOG_LEVEL", "trace")
	os.Setenv("SLRP_LOG_FORMAT", "file")             // TODO: eek, make it better
	os.Setenv("SLRP_LOG_FILE", "$PWD/dist/$APP.log") // TODO: eek, make it better

	main()
}
