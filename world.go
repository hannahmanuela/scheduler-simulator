package slasched

import (
	"fmt"
	"math/rand"
)

const (
	MEM_PER_MACHINE = 1000000

	K_CHOICES_DOWN = 3
	K_CHOICES_UP   = 3

	MIN_QLEN_IDLE_LIST = 8

	TICK_SCHED_THRESHOLD = 0.001

	VERBOSE_USAGE_STATS       = true
	VERBOSE_SCHED_INFO        = false
	VERBOSE_IDEAL_SCHED_INFO  = false
	VERBOSE_HERMOD_SCHED_INFO = false
	VERBOSE_EDF_SCHED_INFO    = false
)

const SEED = 12345

var r = rand.New(rand.NewSource(SEED))

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

func (lbt LBType) newLB(numMachines int, numCores int, metaIter int, nGSSs int, currTickPtr *Tftick) LB {
	return []LB{newMineLB(numMachines, numCores, metaIter, nGSSs, currTickPtr), newIdealLB(numMachines, numCores, metaIter, currTickPtr), newHermodLB(numMachines, numCores, metaIter, nGSSs, currTickPtr), newEDFLB(numMachines, numCores, metaIter, currTickPtr)}[lbt]
}

func (lbt LBType) string() string {
	return []string{"mine", "ideal", "hermod", "edf"}[lbt]
}

type World struct {
	currTick      Tftick
	metaIter      int
	mapTickToLoad func(Tftick) int
	currProcNum   int

	LBs []LB

	loadGen LoadGen
}

func newWorld(numMachines int, numCores int, metaIter int, mapTickToLoad func(Tftick) int, nGSSs int, lbsDoing []LBType) *World {

	w := &World{
		currTick:      Tftick(0),
		metaIter:      metaIter,
		mapTickToLoad: mapTickToLoad,
	}

	for _, lbTypeToInclude := range lbsDoing {
		w.LBs = append(w.LBs, lbTypeToInclude.newLB(numMachines, numCores, metaIter, nGSSs, &w.currTick))
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

func (w *World) Tick() {
	numProcs := w.mapTickToLoad(w.currTick)
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
		w.Tick()
	}
}
