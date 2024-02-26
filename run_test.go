package slasched

import (
	"testing"
)

const (
	NTICK = 2
)

func TestSanityCheck(t *testing.T) {
	numMachines := 1
	numCores := 4
	w := newWorld(numMachines, numCores)
	w.app = newSimpleWebsite()
	w.Run(NTICK)

}
