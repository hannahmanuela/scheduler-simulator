package slasched

import (
	"fmt"
	"testing"
)

const (
	N_TICK = 200

	N_GSSs              = 1
	N_MACHINES          = 100
	N_CORES_PER_MACHINE = 8
)

func mapTickToLoad(tick Tftick) int {
	if tick < 50 {
		return 150
	}
	if (tick >= 50) && (tick < 80) {
		return 200
	}
	return 170
}

func TestRunWorld(t *testing.T) {
	emptyFiles()

	for metaIter := 0; metaIter < 1; metaIter += 1 {

		fmt.Printf("-------- Running metaIter %v --------\n", metaIter)

		w := newWorld(N_MACHINES, N_CORES_PER_MACHINE, metaIter, mapTickToLoad, N_GSSs, []LBType{HERMOD, MINE})
		w.Run(N_TICK)
	}

}
