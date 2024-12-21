package slasched

import (
	"fmt"
	"testing"
	"time"
)

const (
	N_TICK = 1000

	N_MACHINES          = 10
	N_CORES_PER_MACHINE = 8

	N_TENANTS = 8

	// this is per tenant
	// TODO: I think next step is to make this more differentiated -- do pareto distribution? has a mean and an avg vaguely, I think
	// is whats pareto distributed the number of procs or their runtime? or both?
	N_PROCS_GEN_PER_TICK_START = 4
	N_PROCS_GEN_PER_TICK_END   = 8
)

func TestRunWorld(t *testing.T) {
	emptyFiles()
	for nProcsToGen := N_PROCS_GEN_PER_TICK_START; nProcsToGen <= N_PROCS_GEN_PER_TICK_END; nProcsToGen += 1 {
		fmt.Printf("---- Running with %v procs per ticks ----\n", nProcsToGen)
		w := newWorld(N_MACHINES, N_CORES_PER_MACHINE, nProcsToGen, N_TENANTS)
		time.Sleep(100 * time.Millisecond)
		w.Run(N_TICK)
	}

}
