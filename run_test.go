package slasched

import (
	"testing"
)

const (
	NTICK = 100
)

func TestSanityCheck(t *testing.T) {
	numMachines := 20
	w := newWorld(numMachines)
	w.app = newSimpleWebsite()
	w.Run(NTICK)

}
