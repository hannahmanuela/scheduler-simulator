package slasched

import (
	"fmt"
	"math/rand"
)

const (
	MEM_PER_MACHINE = 512000

	IDLE_HEAP_THRESHOLD = 1

	VERBOSE_SCHED_INFO       = false
	VERBOSE_USAGE_STATS      = true
	VERBOSE_IDEAL_SCHED_INFO = false
)

const SEED = 12345

var r = rand.New(rand.NewSource(SEED))

type World struct {
	currTick      Tftick
	numProcsToGen int
	currProcNum   int
	machines      map[Tid]*Machine
	idealDC       *IdealDC
	idealMultiQ   MultiQueue
	gs            *GlobalSched
	tenants       []*Ttenant
}

func newWorld(numMachines int, numCores int, nGenPerTick int, numTenants int) *World {
	w := &World{
		currTick:      Tftick(0),
		machines:      map[Tid]*Machine{},
		numProcsToGen: nGenPerTick,
		idealMultiQ:   NewMultiQ(),
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
		w.gs.multiq.enq(provProc)
		copyForIdeal := newProvProc(Tid(w.currProcNum), w.currTick, up)
		w.idealMultiQ.enq(copyForIdeal)
		w.currProcNum += 1
	}
	return len(userProcs)
}

// this needs to model placement ordering like GS does...
func (w *World) placeProcsIdeal() {

	toReq := make([]*Proc, 0)

	toWrite := fmt.Sprintf("q before placing procs: %v \n", w.idealMultiQ.qMap)
	logWrite(IDEAL_SCHED, toWrite)

	p := w.idealMultiQ.deq(w.currTick)

	for p != nil {
		placed := w.idealDC.potPlaceProc(p)

		if !placed {
			toReq = append(toReq, p)
		}
		p = w.idealMultiQ.deq(w.currTick)
	}

	for _, p := range toReq {
		w.idealMultiQ.enq(p)
	}

}

func (w *World) Tick(numProcs int) {
	w.genLoad(numProcs)

	w.placeProcsIdeal()
	w.gs.placeProcs()

	for _, m := range w.machines {
		m.sched.tick()
	}
	w.idealDC.tick()

	w.currTick += 1
}

func (w *World) Run(nTick int) {
	for i := 0; i < nTick; i++ {
		w.Tick(w.numProcsToGen)
	}
	fmt.Printf("num found idle: %v, num used k choices: %v\n", w.gs.nFoundIdle, w.gs.nUsedKChoices)
}
