package slasched

import (
	"fmt"
	"math"
)

const (
	MAX_MEM                      = 64000 // the amount of memory every machine (currently machine=core) will have, in MB
	SCHEDULER_SLA_INCREMENT_SIZE = 10
	ARRIVAL_RATE                 = 2   // number of procs per tick
	THRESHOLD_MEM_USG_MIN        = 0.4 // avg memory usage below which we will remove a machine
	THRESHOLD_NUM_TICKS_MIN      = 3   // avg number of ticks on a machine below which we will remove a machine
	THRESHOLD_TICKS_AHEAD_MAX    = 1.5 // if the "best option" machine has more than this many ticks ahead, take a new machine
	THRESHOLD_MEM_USG_MAX        = 0.9 // if the "best option" machine has higher than this memory usage, take a new machine

	VERBOSE_SCHEDULER           = false
	VERBOSE_WORLD               = false
	VERBOSE_LB                  = false
	VERBOSE_PROC                = false
	VERBOSE_MACHINES            = false
	VERBOSE_LB_STATS            = true
	VERBOSE_SCHED_STATS         = true
	VERBOSE_WORLD_STATS         = true
	VERBOSE_MACHINE_USAGE_STATS = true
)

type World struct {
	currTick Ttick
	machine  *Machine
	app      Website
}

func newWorld() *World {
	w := &World{}
	w.machine = newMachine(SHINJUKU, Tmid(0))
	return w
}

func (w *World) String() string {
	str := "machines: \n"
	str += "   " + w.machine.String()
	return str
}

func (w *World) genLoad(nProcs int) int {
	userProcs := w.app.genLoad(nProcs)
	sumTicksAdded := Tftick(0)
	for _, up := range userProcs {
		sumTicksAdded += up.actualComp
		provProc := newProvProc(w.currTick, up)
		fmt.Printf("adding: %v, %v, %v, %v, %v, 0\n", w.currTick, w.machine.mid, provProc.procInternals.procType, float64(provProc.procInternals.sla), float64(provProc.procInternals.actualComp))
		w.machine.sched.getQ().enq(provProc)
	}
	if VERBOSE_WORLD {
		fmt.Printf("num ticks added this round: %v\n", sumTicksAdded)
	}
	return len(userProcs)
}

func (w *World) printAllProcs() {
	for _, p := range w.machine.sched.getQ().q {
		fmt.Printf("current: %v, %v, %v, %v, %v\n", w.currTick, w.machine.mid, float64(p.procInternals.sla), float64(p.procInternals.actualComp), float64(p.procInternals.compDone))
	}
}

func (w *World) printTickStats() {
	fmt.Printf("usage: %v, %v, 1, %.2f, %.2f\n", w.currTick, w.machine.mid, math.Abs(float64(w.machine.sched.getTicksUnusedLastTick())), w.machine.sched.memUsage())
}

func (w *World) Tick(numProcsKilled int, numProcsOverSLA_TN int, numProcsOverSLA_FN int) {
	w.currTick += 1
	if VERBOSE_LB_STATS {
		w.printAllProcs()
	}
	// enqueues things into the procq
	w.genLoad(ARRIVAL_RATE)
	// dequeues things from procq to machines based on their util
	if VERBOSE_MACHINES {
		fmt.Printf("after getprocs: %v\n", w)
	}
	// run machine for a tick
	w.machine.sched.tick()

	if VERBOSE_MACHINES {
		fmt.Printf("after compute: %v\n", w)
	}
	if VERBOSE_MACHINE_USAGE_STATS {
		w.printTickStats()
	}

	// if VERBOSE_WORLD {
	// 	fmt.Printf("==============>>>>> TICK %v DONE <<<<<==============\n", w.currTick)
	// 	fmt.Printf("num procs killed this tick %v\n", w.lb.numProcsKilled-numProcsKilled)
	// 	fmt.Printf("num procs over sla TN this tick %v\n", w.lb.numProcsOverSLA_TN-numProcsOverSLA_TN)
	// 	fmt.Printf("num procs over sla FN this tick %v\n \n", w.lb.numProcsOverSLA_FN-numProcsOverSLA_FN)
	// }
}

func (w *World) Run(nTick int) {
	numProcsKilled := 0
	numProcsOverSLA_TN := 0
	numProcsOverSLA_FN := 0
	for i := 0; i < nTick; i++ {
		w.Tick(numProcsKilled, numProcsOverSLA_TN, numProcsOverSLA_FN)
	}
}
