package slasched

import (
	"fmt"
)

const (
	MAX_MEM                      = 12000 // the amount of memory every machine (currently machine=core) will have, in MB
	SCHEDULER_SLA_HISTOGRAM_BASE = 2     // the base of the exponential histogram that procs are placed into in making load balancing decisions
	ARRIVAL_RATE                 = 1     // number of procs per tick per machine
	TARGET_PRESSURE_MIN          = 0     // this is the lower end of the target pressure for machines
	TARGET_PRESSURE_MAX          = 0.5

	VERBOSE_SCHEDULER = false
	VERBOSE_WORLD     = false
	VERBOSE_LB        = false
	VERBOSE_PROC      = false
	VERBOSE_MACHINES  = false
	VERBOSE_STATS     = true
)

type World struct {
	currTick Ttick
	machines []*Machine
	lb       *LoadBalancer
	app      Website
}

func newWorld(numMachines int) *World {
	w := &World{}
	w.machines = make([]*Machine, numMachines)
	lbMachineConn := make(chan *MachineMessages)
	for i := 0; i < numMachines; i++ {
		mid := Tmid(i)
		w.machines[i] = newMachine(mid, lbMachineConn)
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
		for _, p := range m.sched.q.q {
			fmt.Printf("current: %v, %v, %v, %v, %v\n", w.currTick, m.mid, float64(p.procInternals.sla), float64(p.procInternals.actualComp), float64(p.procInternals.compDone))
		}
	}
}

// returns max, min, avd
// func (w *World) getComputePressureStats() (float64, float64, float64) {
// 	maxVal := 0.0
// 	minVal := math.Inf(1)
// 	sum := 0.0
// 	num := 0
// 	for _, m := range w.machines {
// 		press := m.sched.getComputePressure()
// 		sum += press
// 		num += 1
// 		if press > maxVal {
// 			maxVal = press
// 		}
// 		if press < minVal {
// 			minVal = press
// 		}
// 	}
// 	return maxVal, minVal, (sum / float64(num))
// }

func (w *World) Tick(numProcsKilled int, numProcsOverSLA_TN int, numProcsOverSLA_FN int) (int, int, int) {
	w.currTick += 1
	if VERBOSE_STATS {
		w.printAllProcs()
	}
	// enqueues things into the procq
	w.genLoad(int(ARRIVAL_RATE * float64(len(w.machines))))
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
	if VERBOSE_WORLD {
		fmt.Printf("==============>>>>> TICK %v DONE <<<<<==============\n", w.currTick)
		fmt.Printf("num procs killed this tick %v\n", w.lb.numProcsKilled-numProcsKilled)
		fmt.Printf("num procs over sla TN this tick %v\n", w.lb.numProcsOverSLA_TN-numProcsOverSLA_TN)
		fmt.Printf("num procs over sla FN this tick %v\n \n", w.lb.numProcsOverSLA_FN-numProcsOverSLA_FN)
	}
	return w.lb.numProcsKilled, w.lb.numProcsOverSLA_TN, w.lb.numProcsOverSLA_FN
	// min, max, avg := w.getComputePressureStats()
	// fmt.Printf("compute pressures: min %v, max %v, avg %v\n", min, max, avg)
}

func (w *World) Run(nTick int) {
	numProcsKilled := 0
	numProcsOverSLA_TN := 0
	numProcsOverSLA_FN := 0
	for i := 0; i < nTick; i++ {
		numProcsKilled, numProcsOverSLA_TN, numProcsOverSLA_FN = w.Tick(numProcsKilled, numProcsOverSLA_TN, numProcsOverSLA_FN)
	}
}
