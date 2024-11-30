package slasched

import (
	"testing"
	"time"
)

const (
	N_TICK                     = 1000
	N_MACHINES                 = 10
	N_CORES_PER_MACHINE        = 8
	N_PROCS_GEN_PER_TICK_START = 80
	N_PROCS_GEN_PER_TICK_END   = 82
)

// func TestRunWorld(t *testing.T) {
// 	emptyFiles()
// 	w := newWorld(N_MACHINES, N_CORES_PER_MACHINE)
// 	w.app = newSimpleWebsite()
// 	time.Sleep(100 * time.Millisecond)
// 	w.Run(N_TICK, N_PROCS_GEN_PER_TICK)
// }

func TestRunWorld(t *testing.T) {
	emptyFiles()
	for nProcsToGen := N_PROCS_GEN_PER_TICK_START; nProcsToGen <= N_PROCS_GEN_PER_TICK_END; nProcsToGen++ {
		w := newWorld(N_MACHINES, N_CORES_PER_MACHINE, nProcsToGen)
		w.app = newSimpleWebsite()
		time.Sleep(100 * time.Millisecond)
		w.Run(N_TICK)
	}

}
