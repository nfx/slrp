package qa

import (
	"log"
	"os"
	"testing"

	"github.com/mitchellh/go-ps"
)

func InDebug() bool {
	pid := os.Getppid()
	// make syscall only once
	processes, err := ps.Processes()
	if err != nil {
		log.Printf("[ERROR] failed to get processes: %s", err)
		return false
	}
	for {
		if pid == 0 {
			return false
		}
		for i := 0; i < len(processes); i++ {
			p := processes[i]
			if p.Pid() != pid {
				continue
			}
			if p.Executable() == "dlv" {
				return true
			}
			pid = p.PPid()
			break
		}
	}
}

func RunOnlyInDebug(t *testing.T) {
	if !InDebug() {
		t.Skipf("%s is debug-only test", t.Name())
	}
}
