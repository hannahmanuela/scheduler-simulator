package slasched

import (
	"math/rand"
)

const (
	MEM_PER_MACHINE = 512000

	IDLE_HEAP_THRESHOLD = 1

	VERBOSE_PROC_PRINTS      = false
	VERBOSE_SCHED_INFO       = false
	VERBOSE_USAGE_STATS      = true
	VERBOSE_IDEAL_SCHED_INFO = true
)

const SEED = 12345

var r = rand.New(rand.NewSource(SEED))

type World struct {
	currTick      Tftick
	numProcsToGen int
	currProcNum   int
	machines      map[Tid]*Machine
	idealDC       *IdealDC
	gs            *GlobalSched // TODO: actually, I think this should be a hashring or some sort of auto sharding thing
	tenants       []*Ttenant
}

func newWorld(numMachines int, numCores int, nGenPerTick int, numTenants int) *World {
	w := &World{
		currTick:      Tftick(0),
		machines:      map[Tid]*Machine{},
		numProcsToGen: nGenPerTick,
	}
	w.tenants = make([]*Ttenant, numTenants)
	for tid := 0; tid < numTenants; tid++ {
		w.tenants[tid] = newTenant()
	}
	w.idealDC = newIdealDC(numMachines*numCores, Tmem(numMachines*MEM_PER_MACHINE), &w.currTick, nGenPerTick)
	idleHeap := &IdleHeap{
		heap: &MinHeap{},
	}
	for i := 0; i < numMachines; i++ {
		mid := Tid(i)
		w.machines[Tid(i)] = newMachine(mid, idleHeap, numCores, &w.currTick, nGenPerTick)
	}
	w.gs = newGolbalSched(w.machines, &w.currTick, nGenPerTick, idleHeap, w.idealDC)
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
	userProcs := make([]*ProcInternals, 0)
	for _, tn := range w.tenants {
		userProcs = append(userProcs, tn.genLoad(w.numProcsToGen)...)
	}
	for _, up := range userProcs {
		provProc := newProvProc(Tid(w.currProcNum), w.currTick, up)
		w.currProcNum += 1
		w.gs.putProc(provProc)
	}
	return len(userProcs)
}

func (w *World) computeIdeal() {
	// for _, m := range w.machines {
	// 	m.sched.tick()
	// }
	w.idealDC.tick()
}

func (w *World) printAllProcs() {
	for _, m := range w.machines {
		m.sched.printAllProcs()
	}
}

func (w *World) Tick(numProcs int) {
	w.printAllProcs()
	// enqueues things into the procq
	w.genLoad(numProcs)
	// dequeues things from procq to machines
	w.gs.placeProcsIdeal()
	// runs each machine for a tick
	w.computeIdeal()
	w.currTick += 1
}

func (w *World) Run(nTick int) {
	for i := 0; i < nTick; i++ {
		w.Tick(w.numProcsToGen)
	}
}
