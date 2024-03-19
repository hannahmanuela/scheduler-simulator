package slasched

import (
	"testing"
	"time"
)

const (
	NTICK = 1000
)

// func TestBucketRange(t *testing.T) {
// 	lastVal := 0.0
// 	currTest := 0.001
// 	for currTest < 50 {
// 		currVal := getRangeBottomFromSLA(Tftick(currTest))
// 		if currVal != lastVal {
// 			fmt.Println(currVal)
// 			lastVal = currVal
// 		}
// 		currTest += 0.001
// 	}
// }

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
