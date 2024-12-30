package slasched

import (
	"math/rand"
)

const (
	MEM_PER_MACHINE = 64000

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

	idealLB *IdealLB
	mineLB  *MineLB
	// hermodLB HermodLB

	loadGen LoadGen
}

func newWorld(numMachines int, numCores int, nGenPerTick int, nGSSs int) *World {

	w := &World{
		currTick:      Tftick(0),
		numProcsToGen: nGenPerTick,
	}

	w.mineLB = newMineLB(numMachines, numCores, nGenPerTick, nGSSs, &w.currTick)
	w.idealLB = newIdealLB(numMachines, numCores, nGenPerTick, &w.currTick)
	// TODO: hermod lb

	w.loadGen = newLoadGen()

	return w
}

func (w *World) genLoad(nProcs int) []*ProcInternals {

	userProcs := w.loadGen.genLoad(nProcs)

	for _, up := range userProcs {
		provProc := newProvProc(Tid(w.currProcNum), w.currTick, up)
		w.mineLB.placeProc(provProc)

		copyForIdeal := newProvProc(Tid(w.currProcNum), w.currTick, up)
		w.idealLB.placeProc(copyForIdeal)

		// TODO: hermod

		w.currProcNum += 1
	}
	return userProcs
}

func (w *World) Tick(numProcs int) {
	w.genLoad(numProcs)

	w.mineLB.placeProcs()
	w.idealLB.placeProcs()
	// TODO: hermod

	w.mineLB.tick()
	w.idealLB.tick()
	// TODO: hermod

	w.currTick += 1
}

func (w *World) Run(nTick int) {
	for i := 0; i < nTick; i++ {
		w.Tick(w.numProcsToGen)
	}
}
