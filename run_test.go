package slasched

import (
	"testing"
)

const (
	NTICK = 300
)

func TestSanityCheck(t *testing.T) {
	numMachines := 2
	numCores := 4
	emptyFiles()
	w := newWorld(numMachines, numCores)
	w.app = newSimpleWebsite()
	w.Run(NTICK)

}
