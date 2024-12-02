package slasched

import (
	"fmt"
	"math/rand"
)

const (
	MAX_MEM_PER_MACHINE = 32000 // the amount of memory every core will have, in MB

	IDLE_HEAP_THRESHOLD = 1

	VERBOSE_PROC_PRINTS      = false
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
	gs            *GlobalSched
	app           Website
}

func newWorld(numMachines int, numCores int, nGenPerTick int) *World {
	w := &World{
		currTick:      Tftick(0),
		machines:      map[Tid]*Machine{},
		numProcsToGen: nGenPerTick,
	}
	w.idealDC = newIdealDC(numMachines*numCores, &w.currTick, nGenPerTick)
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
	userProcs := w.app.genLoad(nProcs)
	for _, up := range userProcs {
		provProc := newProvProc(Tid(w.currProcNum), w.currTick, up)
		w.currProcNum += 1
		w.gs.putProc(provProc)
	}
	return len(userProcs)
}

func (w *World) compute() {
	for _, m := range w.machines {
		m.sched.tick()
	}
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
	w.gs.placeProcs()
	// runs each machine for a tick
	w.compute()
	w.currTick += 1
}

func (w *World) Run(nTick int) {
	for i := 0; i < nTick; i++ {
		w.Tick(w.numProcsToGen)
	}
	fmt.Printf(" %v: idle \n %v: k-choices \n", w.gs.numFoundIdle, w.gs.numUsedKChoices)
}
