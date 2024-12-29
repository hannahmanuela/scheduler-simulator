package slasched

import (
	"container/heap"
	"fmt"
	"math"
	"os"
)

type Tmem int
type Tftick float64

func (f Tftick) String() string {
	return fmt.Sprintf("%.2f", f)
}

func mapPriorityToPctToGen(priority int) int {
	return []int{35, 25, 2, 15, 5}[priority]
}

func mapPriorityToDollars(priority int) float32 {
	return []float32{0.3, 0.7, 1.0, 1.5, 2}[priority]
}

func genRandPriority() int {

	sample := r.Intn(100)
	currSum := 0

	for prio := 0; prio < N_PRIORITIES; prio++ {
		currSum += mapPriorityToPctToGen(prio)
		if sample < currSum {
			return prio
		}
	}

	return N_PRIORITIES - 1
}

type PrintType int

const (
	PROCS_DONE PrintType = iota
	IDEAL_PROCS_DONE
	SCHED
	USAGE
	IDEAL_USAGE
	IDEAL_SCHED
)

func (pt PrintType) fileName() string {
	return []string{"results/procs_done.txt", "results/ideal_procs_done.txt", "results/sched.txt", "results/usage.txt", "results/ideal_usage.txt", "results/ideal_sched.txt"}[pt]
}

func (pt PrintType) should_print() bool {
	return []bool{VERBOSE_USAGE_STATS, VERBOSE_USAGE_STATS, VERBOSE_SCHED_INFO, VERBOSE_USAGE_STATS, VERBOSE_USAGE_STATS, VERBOSE_IDEAL_SCHED_INFO}[pt]
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
	types := []PrintType{PROCS_DONE, IDEAL_PROCS_DONE, SCHED, USAGE, IDEAL_USAGE, IDEAL_SCHED}

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

func ParetoSample(alpha, xm float64) float64 {
	rnd := r.ExpFloat64()
	return xm * math.Exp(rnd/alpha)
}

func sampleNormal(mu, sigma float64) float64 {
	return r.NormFloat64()*float64(sigma) + float64(mu)
}

func pickRandomElements[T any](list []T, k int) []T {

	if k > len(list) {
		k = len(list)
	}

	if k < len(list) {
		return list
	}

	for i := len(list) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		list[i], list[j] = list[j], list[i]
	}

	return list[:k]
}

func pickRandomElementsMap[K comparable, V any](inpMap map[K]V, k int) map[K]V {

	if k > len(inpMap) {
		k = len(inpMap)
	}

	if k < len(inpMap) {
		return inpMap
	}

	keys := make([]K, 0, len(inpMap))
	for key := range inpMap {
		keys = append(keys, key)
	}

	randKeys := make([]K, k)
	for i := 0; i < k; i++ {
		j := r.Intn(len(inpMap))
		randKeys[i] = keys[j]
	}

	result := make(map[K]V)
	for i := 0; i < k && i < len(keys); i++ {
		key := randKeys[i]
		result[key] = inpMap[key]
	}

	return result
}

func Values[M ~map[K]V, K comparable, V any](m M) []V {
	r := make([]V, 0, len(m))
	for _, v := range m {
		r = append(r, v)
	}
	return r
}
