package slasched

import (
	"fmt"
	"math/rand"
)

const (
	MEM_PER_MACHINE = 512000

	IDLE_HEAP_MEM_THRESHOLD  = 1
	IDLE_HEAP_QLEN_THRESHOLD = 2

	K_CHOICES_DOWN = 3
	K_CHOICES_UP   = 3

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
	GSSs          []*GlobalSched
	roundRobinInd int
	tenants       []*Ttenant
}

func newWorld(numMachines int, numCores int, nGenPerTick int, numTenants int, nGSSs int) *World {

	w := &World{
		currTick:      Tftick(0),
		machines:      map[Tid]*Machine{},
		numProcsToGen: nGenPerTick,
		idealMultiQ:   NewMultiQ(),
		roundRobinInd: 0,
	}

	w.tenants = make([]*Ttenant, numTenants)
	for tid := 0; tid < numTenants; tid++ {
		w.tenants[tid] = newTenant()
	}

	w.idealDC = newIdealDC(numMachines*numCores, Tmem(numMachines*MEM_PER_MACHINE), &w.currTick, nGenPerTick)

	w.GSSs = make([]*GlobalSched, nGSSs)
	idleHeaps := make(map[Tid]*IdleHeap, nGSSs)
	for i := 0; i < nGSSs; i++ {
		idleHeap := &IdleHeap{
			heap: &MinHeap{},
		}
		idleHeaps[Tid(i)] = idleHeap
		w.GSSs[i] = newGolbalSched(i, w.machines, &w.currTick, nGenPerTick, idleHeap, w.idealDC)
	}

	for i := 0; i < numMachines; i++ {
		mid := Tid(i)
		w.machines[Tid(i)] = newMachine(mid, idleHeaps, numCores, &w.currTick, nGenPerTick)
	}

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

		w.GSSs[w.roundRobinInd].multiq.enq(provProc)
		w.roundRobinInd += 1
		if w.roundRobinInd >= len(w.GSSs) {
			w.roundRobinInd = 0
		}

		copyForIdeal := newProvProc(Tid(w.currProcNum), w.currTick, up)
		w.idealMultiQ.enq(copyForIdeal)
		w.currProcNum += 1
	}
	return len(userProcs)
}

// this needs to model placement ordering like GS does...
func (w *World) placeProcsIdeal() {

	toReq := make([]*Proc, 0)

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

	for _, gs := range w.GSSs {
		gs.placeProcs()
		toWrite := fmt.Sprintf("%v, GS %v has heap %v \n", w.currTick, gs.gsId, gs.idleMachines.heap)
		logWrite(SCHED, toWrite)
	}

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

	numIdle := make([]int, len(w.GSSs))
	numKChoices := make([]int, len(w.GSSs))
	for i, gs := range w.GSSs {
		numIdle[i] = gs.nFoundIdle
		numKChoices[i] = gs.nUsedKChoices
	}
	fmt.Printf("num found idle: %v, num used k choices: %v\n", numIdle, numKChoices)
}
