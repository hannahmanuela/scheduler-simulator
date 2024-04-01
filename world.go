package slasched

import (
	"math"
)

const (
	MAX_MEM_PER_MACHINE = 32000 // the amount of memory every core will have, in MB

	TICKS_WAIT_LOAD_CHANGES = 100
	INITIAL_LOAD            = 4
	THRESHOLD_RATIO_MIN     = 0.1 // min max ratio below which we add load
	THRESHOLD_RATIO_MAX     = 0.5 // min max ratio above which we reduce load

	VERBOSE_LB_STATS            = true
	VERBOSE_SCHED_STATS         = true
	VERBOSE_WORLD_STATS         = true
	VERBOSE_MACHINE_USAGE_STATS = true
	VERBOSE_PRESSURE_VALS       = true
)

type World struct {
	currTick        int
	numProcsToGen   int
	lastChangedLoad int
	machines        map[Tid]*Machine
	lb              *LoadBalancer
	app             Website
}

func newWorld(numMachines int) *World {
	w := &World{
		machines:        map[Tid]*Machine{},
		numProcsToGen:   INITIAL_LOAD,
		lastChangedLoad: 0,
	}
	lbMachineConn := make(chan *Message) // channel all machines send on to lb
	machineToLBConns := map[Tid]chan *Message{}
	for i := 0; i < numMachines; i++ {
		mid := Tid(i)
		chanMacheineToLB := make(chan *Message)
		machineToLBConns[mid] = chanMacheineToLB // channel machine receives on
		w.machines[Tid(i)] = newMachine(mid, lbMachineConn, chanMacheineToLB)
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

func (w *World) minMaxRatioTicksPassedToSla() float64 {
	minVal := math.Inf(1)
	for _, m := range w.machines {
		if m.sched.maxRatioTicksPassedToSla() < minVal {
			minVal = m.sched.maxRatioTicksPassedToSla()
		}
	}
	return minVal
}

func (w *World) evalLoad() {
	if w.minMaxRatioTicksPassedToSla() < THRESHOLD_RATIO_MIN && w.currTick-w.lastChangedLoad > TICKS_WAIT_LOAD_CHANGES {
		w.numProcsToGen += 1
		w.lastChangedLoad = w.currTick
	} else if w.minMaxRatioTicksPassedToSla() > THRESHOLD_RATIO_MAX && w.currTick-w.lastChangedLoad > TICKS_WAIT_LOAD_CHANGES {
		w.numProcsToGen -= 1
		w.lastChangedLoad = w.currTick
	}
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
		m.sched.printAllProcs()
	}
}

func (w *World) tickAllProcs() {
	for _, m := range w.machines {
		m.sched.tickAllProcs()
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
	w.tickAllProcs()
}

func (w *World) Run(nTick int) {
	for i := 0; i < nTick; i++ {
		w.evalLoad()
		w.Tick(4)
	}
}
