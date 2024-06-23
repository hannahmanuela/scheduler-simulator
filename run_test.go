package slasched

import (
	"testing"
	"time"
)

const (
	N_TICK               = 20
	N_MACHINES           = 4
	N_PROCS_GEN_PER_TICK = 2
)

func TestRunWorld(t *testing.T) {
	emptyFiles()
	w := newWorld(N_MACHINES)
	w.app = newSimpleWebsite()
	// wait for channels to set up, etc
	time.Sleep(100 * time.Millisecond)
	w.Run(N_TICK, N_PROCS_GEN_PER_TICK)
}
