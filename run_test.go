package slasched

import (
	"testing"
)

const (
	NTICK = 100
)

func TestSanityCheck(t *testing.T) {
	w := newWorld()
	w.app = newSimpleWebsite()
	w.Run(NTICK)

}
