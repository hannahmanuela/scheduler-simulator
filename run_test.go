package slasched

import (
	"testing"
	"time"
)

const (
	NTICK            = 1000
	NMACHINES        = 10
	NPROCSGENPERTICK = 5
)

func TestRunWorld(t *testing.T) {
	emptyFiles()
	w := newWorld(NMACHINES)
	w.app = newSimpleWebsite()
	// wait for channels to set up, etc
	time.Sleep(100 * time.Millisecond)
	w.Run(NTICK, NPROCSGENPERTICK)
}
