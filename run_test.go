package slasched

import "testing"

const (
	NTICK = 3
)

func TestSanityCheck(t *testing.T) {
	numMachines := 1
	numCores := 1
	w := newWorld(numMachines, numCores)
	w.app = newSimpleWebsite(numMachines)
	for i := 0; i < NTICK; i++ {
		w.Tick()
	}
}
