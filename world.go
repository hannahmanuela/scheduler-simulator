package slasched

import (
	"fmt"
	"math"
)

const (
	MAX_MEM_PER_CORE = 11000 // the amount of memory every core will have, in MB

	VERBOSE_LB_STATS            = true
	VERBOSE_SCHED_STATS         = true
	VERBOSE_WORLD_STATS         = true
	VERBOSE_MACHINE_USAGE_STATS = true
)

type World struct {
	currTick Ttick
	machines map[Tid]*Machine
	lb       *LoadBalancer
	app      Website
}

func newWorld(numMachines int, numCores int) *World {
	w := &World{}
	w.machines = map[Tid]*Machine{}
	lbMachineConn := make(chan *Message) // channel all machines send on to lb
	machineToLBConns := map[Tid]chan *Message{}
	for i := 0; i < numMachines; i++ {
		mid := Tid(i)
		chanMacheineToLB := make(chan *Message)
		machineToLBConns[mid] = chanMacheineToLB // channel machine receives on
		w.machines[Tid(i)] = newMachine(mid, numCores, lbMachineConn, chanMacheineToLB)
	}
	w.lb = newLoadBalancer(w.machines, machineToLBConns, lbMachineConn)
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
	return len(userProcs)
}

func (w *World) compute() {
	for _, m := range w.machines {
		m.sched.tick()
	}
}

func (w *World) printAllProcs() {
	for _, m := range w.machines {
		for _, p := range m.sched.q.getQ() {
			toWrite := fmt.Sprintf("%v, %v, %v, %v, %v\n", w.currTick, m.mid,
				float64(p.procInternals.sla), float64(p.procInternals.actualComp), float64(p.compUsed()))
			logWrite(CURR_PROCS, toWrite)
		}
	}
}

func (w *World) printTickStats() {
	for _, m := range w.lb.machines {
		for _, core := range m.sched.coreScheds {
			toWrite := fmt.Sprintf("%v, %v, %v, 1, %.2f, %.2f\n", w.currTick, m.mid, core.coreId,
				math.Abs(float64(core.ticksUnusedLastTick)), core.memUsage())
			logWrite(USAGE, toWrite)
		}
	}
}

func (w *World) Tick(numProcs int) {
	w.currTick += 1
	if VERBOSE_LB_STATS {
		w.printAllProcs()
	}
	// enqueues things into the procq
	w.genLoad(numProcs)
	// dequeues things from procq to machines
	w.lb.placeProcs()
	// runs each machine for a tick
	w.compute()
	if VERBOSE_MACHINE_USAGE_STATS {
		w.printTickStats()
	}
}

func (w *World) Run(nTick int) {
	for i := 0; i < nTick; i++ {
		w.Tick(6)
	}
}
