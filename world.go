package slasched

import (
	"fmt"
	"math/rand"
)

const (
	MEM_PER_MACHINE = 64000

	IDLE_HEAP_MEM_THRESHOLD  = 1
	IDLE_HEAP_QLEN_THRESHOLD = 2

	K_CHOICES_DOWN = 3
	K_CHOICES_UP   = 3

	VERBOSE_USAGE_STATS       = true
	VERBOSE_SCHED_INFO        = false
	VERBOSE_IDEAL_SCHED_INFO  = false
	VERBOSE_HERMOD_SCHED_INFO = false
	VERBOSE_EDF_SCHED_INFO    = false
)

const SEED = 12345

var r = rand.New(rand.NewSource(SEED))

type World struct {
	currTick      Tftick
	numProcsToGen int
	currProcNum   int

	LBs []LB

	loadGen LoadGen
}

type LB interface {
	placeProcs()
	tick()
	enqProc(*Proc)
}

type LBType int

const (
	MINE LBType = iota
	IDEAL
	HERMOD
	EDF
)

func (lbt LBType) newLB(numMachines int, numCores int, nGenPerTick int, nGSSs int, currTickPtr *Tftick) LB {
	return []LB{newMineLB(numMachines, numCores, nGenPerTick, nGSSs, currTickPtr), newIdealLB(numMachines, numCores, nGenPerTick, currTickPtr), newHermodLB(numMachines, numCores, nGenPerTick, nGSSs, currTickPtr), newEDFLB(numMachines, numCores, nGenPerTick, currTickPtr)}[lbt]
}

func (lbt LBType) string() string {
	return []string{"mine", "ideal", "hermod", "edf"}[lbt]
}

func newWorld(numMachines int, numCores int, nGenPerTick int, nGSSs int, lbsDoing []LBType) *World {

	w := &World{
		currTick:      Tftick(0),
		numProcsToGen: nGenPerTick,
	}

	for _, lbTypeToInclude := range lbsDoing {
		w.LBs = append(w.LBs, lbTypeToInclude.newLB(numMachines, numCores, nGenPerTick, nGSSs, &w.currTick))
		fmt.Printf("making lb of type %v\n", lbTypeToInclude.string())
	}

	w.loadGen = newLoadGen()

	return w
}

func (w *World) genLoad(nProcs int) []*ProcInternals {

	userProcs := w.loadGen.genLoad(nProcs)

	for _, up := range userProcs {

		for _, lb := range w.LBs {
			provProc := newProvProc(Tid(w.currProcNum), w.currTick, up)
			lb.enqProc(provProc)
		}

		w.currProcNum += 1
	}
	return userProcs
}

func (w *World) Tick(numProcs int) {
	w.genLoad(numProcs)

	for _, lb := range w.LBs {
		lb.placeProcs()
	}

	for _, lb := range w.LBs {
		lb.tick()
	}

	w.currTick += 1
}

func (w *World) Run(nTick int) {
	for i := 0; i < nTick; i++ {
		w.Tick(w.numProcsToGen)
	}
}
