package slasched

import (
	"fmt"
	"testing"
)

const (
	N_TICK = 100

	N_GSSs              = 1
	N_MACHINES          = 5
	N_CORES_PER_MACHINE = 8

	// this is overall
	N_PROCS_GEN_PER_TICK_START = 10
	N_PROCS_GEN_PER_TICK_END   = 10
)

func TestRunWorld(t *testing.T) {
	emptyFiles()
	for nProcsToGen := N_PROCS_GEN_PER_TICK_START; nProcsToGen <= N_PROCS_GEN_PER_TICK_END; nProcsToGen += 10 {

		fmt.Printf("---- Running with %v procs per ticks ----\n", nProcsToGen)

		w := newWorld(N_MACHINES, N_CORES_PER_MACHINE, nProcsToGen, N_GSSs, []LBType{MINE})
		w.Run(N_TICK)
	}

}
