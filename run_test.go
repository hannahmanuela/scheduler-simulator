package slasched

import (
	"testing"
	"time"
)

const (
	NTICK = 1000
)

func TestRunWorld(t *testing.T) {
	numMachines := 8
	emptyFiles()
	w := newWorld(numMachines)
	w.app = newSimpleWebsite()
	// wait for channels to set up, etc
	time.Sleep(100 * time.Millisecond)
	w.Run(NTICK)
}
