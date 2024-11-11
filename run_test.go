package slasched

import (
	"testing"
	"time"
)

const (
	N_TICK               = 100
	N_MACHINES           = 10
	N_PROCS_GEN_PER_TICK = 4
)

func TestRunWorld(t *testing.T) {
	emptyFiles()
	w := newWorld(N_MACHINES)
	w.app = newSimpleWebsite()
	// wait for channels to set up, etc
	time.Sleep(100 * time.Millisecond)
	w.Run(N_TICK, N_PROCS_GEN_PER_TICK)
}
