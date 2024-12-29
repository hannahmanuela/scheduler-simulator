package slasched

import (
	"fmt"
	"math/rand"
)

const (
	MEM_PER_MACHINE = 128000

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
	GSSs          []*GlobalSched
	roundRobinInd int
	loadGen       LoadGen
}

func newWorld(numMachines int, numCores int, nGenPerTick int, nGSSs int) *World {

	w := &World{
		currTick:      Tftick(0),
		machines:      map[Tid]*Machine{},
		numProcsToGen: nGenPerTick,
		roundRobinInd: 0,
	}

	w.loadGen = newLoadGen()

	w.GSSs = make([]*GlobalSched, nGSSs)
	idleHeaps := make(map[Tid]*IdleHeap, nGSSs)
	for i := 0; i < nGSSs; i++ {
		idleHeap := &IdleHeap{
			heap: &MinHeap{},
		}
		idleHeaps[Tid(i)] = idleHeap
		w.GSSs[i] = newGolbalSched(i, w.machines, &w.currTick, nGenPerTick, idleHeap)
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

	userProcs := w.loadGen.genLoad(nProcs)

	for _, up := range userProcs {
		provProc := newProvProc(Tid(w.currProcNum), w.currTick, up)

		w.GSSs[w.roundRobinInd].multiq.enq(provProc)
		w.roundRobinInd += 1
		if w.roundRobinInd >= len(w.GSSs) {
			w.roundRobinInd = 0
		}

		w.currProcNum += 1
	}
	return len(userProcs)
}

func (w *World) Tick(numProcs int) {
	w.genLoad(numProcs)

	for _, gs := range w.GSSs {
		gs.placeProcs()
		toWrite := fmt.Sprintf("%v, GS %v has heap %v \n", w.currTick, gs.gsId, gs.idleMachines.heap)
		logWrite(SCHED, toWrite)
	}

	for _, m := range w.machines {
		m.sched.tick()
	}

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
