package slasched

import (
	"testing"
)

const (
	NTICK = 100
)

func TestSanityCheck(t *testing.T) {
	numMachines := 1
	numCoresPerMachine := 8
	w := newWorld(numMachines, numCoresPerMachine)
	w.app = newSimpleWebsite()
	w.Run(NTICK)

}
