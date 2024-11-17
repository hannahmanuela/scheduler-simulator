package slasched

import (
	"testing"
	"time"
)

const (
	N_TICK               = 1000
	N_MACHINES           = 10
	N_CORES_PER_MACHINE  = 8
	N_PROCS_GEN_PER_TICK = 30
)

func TestRunWorld(t *testing.T) {
	emptyFiles()
	w := newWorld(N_MACHINES, N_CORES_PER_MACHINE)
	w.app = newSimpleWebsite()
	// wait for channels to set up, etc
	time.Sleep(100 * time.Millisecond)
	w.Run(N_TICK, N_PROCS_GEN_PER_TICK)
}
