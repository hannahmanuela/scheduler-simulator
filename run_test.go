package slasched

import (
	"testing"
	"time"
)

const (
	NTICK = 1000
)

func TestSanityCheck(t *testing.T) {
	numMachines := 2
	numCores := 4
	emptyFiles()
	w := newWorld(numMachines, numCores)
	w.app = newSimpleWebsite()
	// wait for channels to set up, etc
	time.Sleep(100 * time.Millisecond)
	w.Run(NTICK)
}
