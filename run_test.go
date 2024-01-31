package slasched

import (
	"fmt"
	"testing"
)

const (
	NTICK = 200
)

func TestSanityCheck(t *testing.T) {
	numMachines := 100
	w := newWorld(numMachines)
	w.app = newSimpleWebsite(numMachines)
	w.Run(NTICK)

	fmt.Println("---------------")
	fmt.Println("---------------")
	fmt.Println("run done!")
	fmt.Printf("num procs killed: %v\n", w.loadBalancer.numProcsKilled)
	fmt.Printf("num procs over sla TN: %v\n", w.loadBalancer.numProcsOverSLA_TN)
	fmt.Printf("num procs over sla FN: %v\n", w.loadBalancer.numProcsOverSLA_FN)
}
