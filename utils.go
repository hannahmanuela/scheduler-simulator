package slasched

import (
	"fmt"
	"math/rand"
	"os"
)

type Ttick int
type Tmem int

type Tftick float64

func (f Tftick) String() string {
	return fmt.Sprintf("%.3fT", f)
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

func sampleNormal(mu, sigma float64) float64 {
	return rand.NormFloat64()*float64(sigma) + float64(mu)
}
