package slasched

import "testing"

const (
	NTICK = 5
)

func TestSanityCheck(t *testing.T) {
	numMachines := 1
	w := newWorld(numMachines)
	w.app = newSimpleWebsite(numMachines)
	for i := 0; i < NTICK; i++ {
		w.Tick()
	}
}
