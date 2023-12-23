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
	MAX_SERVICE_TIME                         = 10 // in ticks
	MAX_MEM                                  = 10
	PROC_DEVIATION_FROM_SLA_VARIANCE         = 0.5
	PROC_SLA_EXPECTED_BUFFER                 = 0.2 // as a fraction of sla
	AVG_ARRIVAL_RATE_SMALL           float64 = 4   // per tick (with 1 tick per proc)
)

type World struct {
	ntick    Ttick
	machines map[Tmid]*Machine
	procq    *Queue
	rand     *rand.Rand
	app      Website
}

func newWorld(nMachines int) *World {
	w := &World{}
	w.machines = make(map[Tmid]*Machine, nMachines)
	w.procq = &Queue{q: make([]*Proc, 0)}
	for i := 0; i < nMachines; i++ {
		mid := Tmid(i)
		w.machines[mid] = newMachine(mid)
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
	procs := w.app.genLoad(w.rand, w.ntick)
	fmt.Printf("generated %d procs\n", len(procs))
	for _, p := range procs {
		fmt.Printf("enqing proc \n")
		w.procq.enq(p)
	}
}

func (w *World) getProc(n Tmem) *Proc {
	// TODO: make this more differentiated
	return w.procq.deq()
}

// TODO: make this take into account not only mem, but also buffers to SLA
func (w *World) placeProcs() {
	capacityAvailable := true
	i := 0
	for capacityAvailable {
		c := false
		for _, m := range w.machines {
			memUsed := m.schedd.memUsed()
			if memUsed < m.schedd.totMem {
				if p := w.getProc(m.schedd.totMem - memUsed); p != nil {
					c = true
					m.schedd.q.enq(p)
				}
			}
		}
		capacityAvailable = c
		i += 1
	}
}

func (w *World) compute() {
	for _, m := range w.machines {
		m.schedd.tick()
	}
}

func (w *World) Tick() {
	w.ntick += 1
	// enqueues things into the procq
	w.genLoad()
	// dequeues things from procq to machines based on their util
	w.placeProcs()
	fmt.Printf("after getprocs: %v\n", w)
	// runs each machine for a tick
	w.compute()
	fmt.Printf("after compute: %v\n", w)
}
