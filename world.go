package slasched

import (
	"fmt"
)

const (
	MAX_MEM                      = 12000 // the amount of memory every machine (currently machine=core) will have, in MB
	PROC_MEM_CHANGE_MAX          = 5     // the maximal increase in memory usage a proc can experience when it runs
	PROC_MEM_CHANGE_MIN          = -2    // the actual value is chosen uniform random between min and max
	SCHEDULER_SLA_INCREMENT_SIZE = 0.5   // the increment size that we group slas together when creating histogram of procs on machines
	ARRIVAL_RATE                 = 1     // number of procs per tick per machine
	TARGET_PRESSURE_MIN          = 0     // this is the lower end of the target pressure for machines
	TARGET_PRESSURE_MAX          = 0.5

	VERBOSE_SCHEDULER = false
	VERBOSE_WORLD     = true
	VERBOSE_LB        = false
	VERBOSE_PROC      = false
	VERBOSE_MACHINES  = false
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
	lbMachineConn := make(chan *MachineMessages)
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

func (w *World) genLoad(nProcs int) int {
	userProcs := w.app.genLoad(nProcs)
	for _, up := range userProcs {
		provProc := newProvProc(w.currTick, up)
		w.loadBalancer.putProc(provProc)
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
	// enqueues things into the procq
	w.genLoad(ARRIVAL_RATE * len(w.machines))
	// dequeues things from procq to machines based on their util
	w.loadBalancer.placeProcs()
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
		fmt.Printf("num procs killed this tick %v\n", w.loadBalancer.numProcsKilled-numProcsKilled)
		fmt.Printf("num procs over sla TN this tick %v\n", w.loadBalancer.numProcsOverSLA_TN-numProcsOverSLA_TN)
		fmt.Printf("num procs over sla FN this tick %v\n", w.loadBalancer.numProcsOverSLA_FN-numProcsOverSLA_FN)
	}
	return w.loadBalancer.numProcsKilled, w.loadBalancer.numProcsOverSLA_TN, w.loadBalancer.numProcsOverSLA_FN
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
