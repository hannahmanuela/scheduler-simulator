package main

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	MAX_SERVICE_TIME                 = 10 // in ticks
	MAX_MEM                          = 10
	PROC_DEVIATION_FROM_SLA_VARIANCE = 5
	PROC_SLA_EXPECTED_BUFFER         = 0.2 // as a fraction of sla
)

type World struct {
	ntick    Ttick
	machines map[Tmid]*Machine
	procq    *Queue
	rand     *rand.Rand
	app      Website
	nproc    int
	maxq     int
	avgq     float64
}

func newWorld(nMachines int) *World {
	w := &World{}
	w.machines = make(map[Tmid]*Machine, nMachines)
	w.procq = &Queue{q: make([]*Proc, 0)}
	for i := 0; i < len(w.machines); i++ {
		mid := Tmid(i)
		w.machines[mid] = newMachine(mid)
	}
	w.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	return w
}

func (w *World) String() string {
	str := fmt.Sprintf("%d nproc %v maxq %d avgq %.1f util %.1f%%\n", w.ntick, w.nproc, w.maxq, w.avgq/float64(w.ntick), w.util())
	str += "machines: [\n"
	for _, m := range w.machines {
		str += "  " + m.String() + ",\n"
	}
	str += "  ]\n procQ:" + w.procq.String()
	return str
}

func (w *World) util() float64 {
	u := float64(0)
	for _, m := range w.machines {
		u += m.util()
	}
	return (u / float64(w.ntick)) * float64(100)
}

func (w *World) genLoad() {
	procs := w.app.genLoad(w.rand)
	for _, p := range procs {
		w.nproc += 1
		w.procq.enq(p)
	}
}

func (w *World) getProc(n Tmem) *Proc {
	return w.procq.deq(n)
}

func (w *World) placeProcs() {
	capacityAvailable := true
	i := 0
	for capacityAvailable {
		c := false
		for _, m := range w.machines {
			mem := m.schedd.memUsed()
			if mem < m.schedd.totMem {
				if p := w.getProc(m.schedd.totMem - mem); p != nil {
					c = true
					m.schedd.q.enq(p)
				}
			}
		}
		capacityAvailable = c
		i += 1
	}
	for _, m := range w.machines {
		mem := m.schedd.memUsed()
		if mem < m.schedd.totMem {
			fmt.Printf("WARNING CAPACITY %v\n", m.schedd)
		}
	}
}

func (w *World) compute() {
	for _, m := range w.machines {
		m.schedd.run()
	}
}

func (w *World) zap(mid Tmid) {
	fmt.Printf("zap a proc from machine %v at %v\n", mid, w.ntick)
	for _, m := range w.machines {
		if m.schedd.zap() {
			return
		}
	}
}

func (w *World) utilPerMachine() Ttickmap {
	mutil := make(Ttickmap)
	for mid, m := range w.machines {
		mutil[mid] = Tftick(m.schedd.util)
	}
	return mutil
}

func (w *World) hasWork(mid Tmid) bool {
	if w.machines[mid].schedd.q.qlen() > 0 {
		return true
	}
	return false
}

// func (w *World) zapper() {
// 	mutil := w.utilPerMachine()
// 	// fmt.Printf("rutil %v avg %v h %v hr %v l %v lr %v\n", rutil, avg, h, hr, l, lr)
// 	// TODO: what's this doing? if the lowest used has work, zap the highest used?
// 	// ig comes down to what is zapping
// 	if w.hasWork(lr) {
// 		w.zap(hr)
// 	}
// }

func (w *World) qstat() {
	qlen := 0
	for _, m := range w.machines {
		qlen += m.schedd.q.qlen()
	}
	// TODO: why are we just adding to the avg q??
	w.avgq += float64(qlen)
	if qlen > w.maxq {
		w.maxq = qlen
	}
}

func (w *World) Tick() {
	w.ntick += 1
	// enqueues things into the procq
	w.genLoad()
	fmt.Printf("after gen %v\n", w)
	// dequeues things from procq to machines based on their util
	w.placeProcs()
	fmt.Printf("after getprocs %v\n", w)
	w.compute()
	// w.zapper()
	w.qstat()
	fmt.Printf("after compute %v\n", w)
}
