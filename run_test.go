package slasched

import (
	"fmt"
	"testing"
)

const (
	N_TICK = 100

	N_GSSs              = 4
	N_MACHINES          = 100
	N_CORES_PER_MACHINE = 8

	// this is overall
	N_PROCS_GEN_PER_TICK_START = 190
	N_PROCS_GEN_PER_TICK_END   = 275
)

func TestRunWorld(t *testing.T) {
	emptyFiles()
	for nProcsToGen := N_PROCS_GEN_PER_TICK_START; nProcsToGen <= N_PROCS_GEN_PER_TICK_END; nProcsToGen += 5 {

		fmt.Printf("---- Running with %v procs per ticks ----\n", nProcsToGen)

		w := newWorld(N_MACHINES, N_CORES_PER_MACHINE, nProcsToGen, N_GSSs, []LBType{EDF, IDEAL})
		w.Run(N_TICK)
	}

}
