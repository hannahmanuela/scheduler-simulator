package slasched

import (
	"fmt"
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

	fmt.Println("---------------")
	fmt.Println("---------------")
	fmt.Printf("total num procs killed: %v\n", w.lb.numProcsKilled)
	fmt.Printf("total num procs over sla TN: %v\n", w.lb.numProcsOverSLA_TN)
	fmt.Printf("total num procs over sla FN: %v\n", w.lb.numProcsOverSLA_FN)
}
