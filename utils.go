package slasched

import (
	"container/heap"
	"fmt"
	"os"
)

type Tmem int
type Tftick float64

func (f Tftick) String() string {
	return fmt.Sprintf("%.2fT", f)
}

type PrintType int

const (
	CURR_PROCS PrintType = iota
	ADDED_PROCS
	DONE_PROCS
	SCHED
	USAGE
	SAID_NO
	CREATED_PROCS
	IDEAL_SAID_NO
	IDEAL_USAGE
	IDEAL_SCHED
)

func (pt PrintType) fileName() string {
	return []string{"results/procs_current.txt", "results/procs_added.txt", "results/procs_done.txt", "results/sched.txt", "results/usage.txt", "results/said_no.txt", "results/procs_created.txt", "results/ideal_said_no.txt", "results/ideal_usage.txt", "results/ideal_sched.txt"}[pt]
}

func (pt PrintType) should_print() bool {
	return []bool{VERBOSE_PROC_PRINTS, VERBOSE_PROC_PRINTS, VERBOSE_PROC_PRINTS, VERBOSE_SCHED_INFO, VERBOSE_USAGE_STATS, VERBOSE_USAGE_STATS, VERBOSE_USAGE_STATS, VERBOSE_USAGE_STATS, VERBOSE_USAGE_STATS, VERBOSE_IDEAL_SCHED_INFO}[pt]
}

func logWrite(printType PrintType, toWrite string) {

	if !printType.should_print() {
		return
	}

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
	types := []PrintType{CURR_PROCS, ADDED_PROCS, DONE_PROCS, SCHED, USAGE, SAID_NO, CREATED_PROCS, IDEAL_SAID_NO, IDEAL_USAGE, IDEAL_SCHED}

	for _, t := range types {
		os.Truncate(t.fileName(), 0)
	}

}

func contains(h *MinHeap, value Tid) bool {
	for _, v := range *h {
		if v.machine == value {
			return true
		}
	}
	return false
}

func remove(h *MinHeap, toRemove Tid) {
	for i := 0; i < h.Len(); i++ {
		if (*h)[i].machine == toRemove {
			heap.Remove(h, i)
			break
		}
	}
}

func sampleNormal(mu, sigma float64) float64 {
	return r.NormFloat64()*float64(sigma) + float64(mu)
}

func pickRandomElements[T any](list []T, k int) []T {

	if k > len(list) {
		k = len(list)
	}

	// Use the Fisher-Yates shuffle algorithm to shuffle the list
	for i := len(list) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		list[i], list[j] = list[j], list[i]
	}

	return list[:k]
}

func Values[M ~map[K]V, K comparable, V any](m M) []V {
	r := make([]V, 0, len(m))
	for _, v := range m {
		r = append(r, v)
	}
	return r
}
