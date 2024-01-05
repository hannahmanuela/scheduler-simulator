package slasched

import (
	"fmt"
	"math"
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
	PROC_DEVIATION_FROM_SLA_VARIANCE = 0.5 // variance of procs actual runtime to "expected" runtime (sla - sla * expected buffer)
	PROC_SLA_EXPECTED_BUFFER         = 0.2 // as a fraction of sla
	PROC_SLA_RANGE_MAX               = 5   // the max value that a sla can have - slas will have uniform random value in this range
	SCHEDULER_SLA_INCREMENT_SIZE     = 2   // the increment size that we group slas together when creating histogram of procs on machines
	AVG_ARRIVAL_RATE                 = 5   // per tick per machine (with 1 tick per proc)
)

type World struct {
	currTick Ttick
	machines map[Tmid]*Machine
	procq    *Queue
	app      Website
}

func newWorld(numMachines int) *World {
	w := &World{}
	w.machines = make(map[Tmid]*Machine, numMachines)
	w.procq = &Queue{q: make([]*Proc, 0)}
	for i := 0; i < numMachines; i++ {
		mid := Tmid(i)
		w.machines[mid] = newMachine(mid)
	}
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
	userProcs := w.app.genLoad()
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

func (w *World) placeProcs() {
	p := w.getProc()
	for p != nil {
		var machineToUse *Machine
		minScore := math.Inf(1)
		// place given proc
		for _, m := range w.machines {
			score := m.sched.pressureScore()
			if score < minScore {
				machineToUse = m
				minScore = score
			}
		}
		fmt.Printf("machine %d has score %v, and is thus being used\n", machineToUse.mid, minScore)
		machineToUse.sched.q.enq(p)
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
	fmt.Printf("map for machine 0: %v\n", w.machines[0].sched.makeHistogram())
	// runs each machine for a tick
	w.compute()
	fmt.Printf("after compute: %v\n", w)
}
