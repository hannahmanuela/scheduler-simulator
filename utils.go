package slasched

import (
	"container/heap"
	"fmt"
	"math/rand"
	"os"
)

type Tmem int
type Tftick float64

func (f Tftick) String() string {
	return fmt.Sprintf("%.6fT", f)
}

type PrintType int

const (
	CURR_PROCS PrintType = iota
	ADDED_PROCS
	DONE_PROCS
	SCHED
	USAGE
)

func (pt PrintType) fileName() string {
	return []string{"results/procs_current.txt", "results/procs_added.txt", "results/procs_done.txt", "results/sched.txt", "results/usage.txt"}[pt]
}

func logWrite(printType PrintType, toWrite string) {
	f, err := os.OpenFile(printType.fileName(), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	if _, err = f.WriteString(toWrite); err != nil {
		panic(err)
	}
}

func emptyFiles() {
	types := []PrintType{CURR_PROCS, ADDED_PROCS, DONE_PROCS, SCHED, USAGE}

	for _, t := range types {
		os.Truncate(t.fileName(), 0)
	}

}

func contains(h *MinHeap, value Tid) bool {
	for _, v := range *h {
		if v.machineId == value {
			return true
		}
	}
	return false
}

func remove(h *MinHeap, toRemove Tid) {
	for i := 0; i < h.Len(); i++ {
		if (*h)[i].machineId == toRemove {
			heap.Remove(h, i)
			break
		}
	}
}

func sampleNormal(mu, sigma float64) float64 {
	return rand.NormFloat64()*float64(sigma) + float64(mu)
}

// returns the bottom value of the SLA range in which the passed SLA is
// this is helpful for creating the histogram mapping number of procs in the scheduler to SLA slices
// eg if we are looking at SLAs in an increment size of 2 and this is given 1.5, it will return 0 (since 1.5 would be in the 0-2 range)
func getRangeBottomFromSLA(sla Tftick) float64 {

	// bucketIndex := math.Ceil(math.Log(float64(sla)/BUCKETS_INIT_SIZE) / math.Log(BUCKETS_BASE))
	// lowerBound := math.Pow(BUCKETS_BASE, bucketIndex-1) * BUCKETS_INIT_SIZE
	lowerBound := 0.0

	if sla <= Tftick(2) {
		lowerBound = 0
	} else if sla <= Tftick(5) {
		lowerBound = 2
	} else if sla <= Tftick(10) {
		lowerBound = 5
	} else {
		lowerBound = 10
	}

	return lowerBound
}
