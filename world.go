package slasched

import (
	"fmt"
)

// notes:
// - no sense of waiting for i/o (this is pretty simple: just when scheduling, look only at procs that could run right now)
// - memory usage of procs is random (also simple: otherwise procs need a map from ticksFromStart to the memory usage)
// - machines can only run one proc at a time (simple-ish, scheduler now becomes per-core, and we add a middle layer scheduler that's on the machine but only distributes procs)
// - scheduler pre-schedules everything (rather than making decisions on the fly) (does this need to change? only really if a proc comes in during that time, right?)

const (
	MAX_MEM                      = 20
	PROC_MEM_CHANGE_MAX          = 5   // the maximal increase in memory usage a proc can experience when it runs
	PROC_MEM_CHANGE_MIN          = -2  // the actual value is chosen uniform random between min and max
	SCHEDULER_SLA_INCREMENT_SIZE = 0.5 // the increment size that we group slas together when creating histogram of procs on machines
	AVG_ARRIVAL_RATE             = 3   // per tick per machine
)

type World struct {
	currTick     Ttick
	machines     []*Machine
	loadBalancer *LoadBalancer
	app          Website
}

func newWorld(numMachines int) *World {
	w := &World{}
	w.machines = make([]*Machine, numMachines)
	lbMachineConn := make(chan *Proc)
	for i := 0; i < numMachines; i++ {
		mid := Tmid(i)
		w.machines[i] = newMachine(mid, lbMachineConn)
	}
	w.loadBalancer = newLoadBalancer(w.machines, lbMachineConn)
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
		w.loadBalancer.putProc(provProc)
	}
}

func (w *World) compute() {
	for _, m := range w.machines {
		fmt.Printf(" ----- TICKING MACHINE %v ----- \n ", m.mid)
		m.sched.tick()
	}
}

func (w *World) Tick() {
	w.currTick += 1
	// enqueues things into the procq
	w.genLoad()
	// dequeues things from procq to machines based on their util
	w.loadBalancer.placeProcs()
	fmt.Printf("after getprocs: %v\n", w)
	// runs each machine for a tick
	w.compute()
	fmt.Printf("after compute: %v\n", w)
}
