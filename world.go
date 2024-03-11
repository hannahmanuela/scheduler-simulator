package slasched

import (
	"fmt"
)

const (
	MAX_MEM_PER_CORE             = 32000 // the amount of memory every machine (currently machine=core) will have, in MB
	SCHEDULER_SLA_INCREMENT_SIZE = 10
	ARRIVAL_RATE                 = 8   // number of procs per tick
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
	machines []*Machine
	lb       *LoadBalancer
	app      Website
}

func newWorld(numMachines int, numCorePerMachine int) *World {
	w := &World{}
	w.machines = make([]*Machine, numMachines)
	lbMachineConn := make(chan *MachineMessages)
	for i := 0; i < numMachines; i++ {
		mid := Tid(i)
		w.machines[i] = newMachine(mid, lbMachineConn, numCorePerMachine)
	}
	w.lb = newLoadBalancer(w.machines, lbMachineConn)
	return w
}

func (w *World) String() string {
	str := "machines: \n"
	for _, m := range w.machines {
		str += "   " + m.String()
	}
	return str
}

func (w *World) genLoad(nProcs int) int {
	userProcs := w.app.genLoad(nProcs)
	sumTicksAdded := Tftick(0)
	for _, up := range userProcs {
		sumTicksAdded += up.actualComp
		provProc := newProvProc(w.currTick, up)
		w.lb.putProc(provProc)
	}
	if VERBOSE_WORLD {
		fmt.Printf("num ticks added this round: %v\n", sumTicksAdded)
	}
	return len(userProcs)
}

func (w *World) compute() {
	for _, m := range w.machines {
		if VERBOSE_SCHEDULER {
			fmt.Printf(" ----- TICKING MACHINE %v ----- \n ", m.mid)
		}
		m.sched.tick()
	}
}

func (w *World) printAllProcs() {
	for _, m := range w.machines {
		m.sched.printAllProcs()
	}
}

func (w *World) Tick() {
	if VERBOSE_LB_STATS {
		w.printAllProcs()
	}
	// enqueues things into the procq
	w.genLoad(ARRIVAL_RATE)
	// dequeues things from procq to machines based on their util
	w.lb.placeProcs()
	if VERBOSE_MACHINES {
		fmt.Printf("after getprocs: %v\n", w)
	}
	// runs each machine for a tick
	w.compute()
	if VERBOSE_MACHINES {
		fmt.Printf("after compute: %v\n", w)
	}
	w.currTick += 1
}

func (w *World) Run(nTick int) {
	for i := 0; i < nTick; i++ {
		w.Tick()
	}
}
