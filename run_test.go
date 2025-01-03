package slasched

import (
	"fmt"
	"testing"
)

const (
	N_TICK = 200

	N_GSSs              = 4
	N_MACHINES          = 100
	N_CORES_PER_MACHINE = 8

	// this is overall
	N_PROCS_GEN_PER_TICK_START = 170
	N_PROCS_GEN_PER_TICK_END   = 230
)

func TestRunWorld(t *testing.T) {
	emptyFiles()
	for nProcsToGen := N_PROCS_GEN_PER_TICK_START; nProcsToGen <= N_PROCS_GEN_PER_TICK_END; nProcsToGen += 30 {

		fmt.Printf("---- Running with %v procs per ticks ----\n", nProcsToGen)

		w := newWorld(N_MACHINES, N_CORES_PER_MACHINE, nProcsToGen, N_GSSs)
		w.Run(N_TICK)
	}

}
