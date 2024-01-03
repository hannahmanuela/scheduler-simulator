package slasched

import (
	"fmt"
	"math/rand"
	"time"
)

// notes:
// - only starts counting time passed once proc is placed on machine
// - a machine runs one proc at a time and progress is only measured by
//   how long a proc runs, ie:
//    - does not support any sort of paralelization
//    - does not know about waiting for i/o

const (
	MAX_SERVICE_TIME                 = 10 // in ticks
	MAX_MEM                          = 10
	PROC_DEVIATION_FROM_SLA_VARIANCE = 0.5
	PROC_SLA_EXPECTED_BUFFER         = 0.2 // as a fraction of sla
	AVG_ARRIVAL_RATE                 = 2.5 // per tick (with 1 tick per proc)
	CURR_MACHINE                     = 0
)

type World struct {
	currTick Ttick
	machines map[Tmid]*Machine
	procq    *Queue
	rand     *rand.Rand
	app      Website
	currMid  Tmid
}

func newWorld(numMachines int, numCoresPerMachine int) *World {
	w := &World{}
	w.machines = make(map[Tmid]*Machine, numMachines)
	w.procq = &Queue{q: make([]*Proc, 0)}
	for i := 0; i < numMachines; i++ {
		mid := Tmid(i)
		w.machines[mid] = newMachine(mid, numCoresPerMachine)
	}
	w.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	return w
}

func (w *World) String() string {
	str := "machines: \n"
	for _, m := range w.machines {
		str += "   " + m.String()
	}
	return str
}

func (w *World) genLoad() {
	userProcs := w.app.genLoad(w.rand)
	fmt.Printf("generated %d procs\n", len(userProcs))
	for _, up := range userProcs {
		provProc := newProvProc(w.currTick, up)
		fmt.Printf("enqing proc \n")
		w.procq.enq(provProc)
	}
}

func (w *World) getProc() *Proc {
	return w.procq.deq()
}

// this is currently just round robin
func (w *World) placeProcs() {
	p := w.getProc()
	for p != nil {
		w.machines[w.currMid].sched.takeProc(p)
		w.currMid += 1
		w.currMid = Tmid(int(w.currMid) % len(w.machines))
		p = w.getProc()
	}
}

func (w *World) compute() {
	for _, m := range w.machines {
		m.sched.tick()
	}
}

func (w *World) Tick() {
	w.currTick += 1
	// enqueues things into the procq
	w.genLoad()
	// dequeues things from procq to machines based on their util
	w.placeProcs()
	fmt.Printf("after getprocs: %v\n", w)
	// runs each machine for a tick
	w.compute()
	fmt.Printf("after compute: %v\n", w)
}
