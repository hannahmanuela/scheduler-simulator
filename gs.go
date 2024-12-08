package slasched

import (
	"fmt"
	"math"
)

type TmachineCoreId struct {
	machineId Tid
	coreId    Tid
}

type GlobalSched struct {
	machines        map[Tid]*Machine
	k_choices       int
	idealDC         *IdealDC
	procq           *Queue
	currTickPtr     *Tftick
	nProcGenPerTick int
	numFoundIdle    map[int]int
	numUsedKChoices map[int]int
}

func newGolbalSched(machines map[Tid]*Machine, currTickPtr *Tftick, numGenPerTick int, idealDC *IdealDC) *GlobalSched {
	gs := &GlobalSched{
		machines:        machines,
		k_choices:       int(len(machines) / 3),
		idealDC:         idealDC,
		procq:           newQueue(),
		currTickPtr:     currTickPtr,
		nProcGenPerTick: numGenPerTick,
		numFoundIdle:    make(map[int]int),
		numUsedKChoices: make(map[int]int),
	}

	return gs
}

func (gs *GlobalSched) MachinesString() string {
	str := "machines: \n"
	for _, m := range gs.machines {
		str += "   " + m.String()
	}
	return str
}

func (gs *GlobalSched) placeProcs() {
	// setup
	p := gs.getProc()

	for p != nil {
		// place given proc

		// check that the tenant has enough tokens to run the proc, if not queue it

		toWrite := fmt.Sprintf("%v, %v, %v \n", gs.nProcGenPerTick, int(*gs.currTickPtr), int(p.priority))
		logWrite(CREATED_PROCS, toWrite)

		// place on ideal
		procCopy := newProvProc(p.procId, *gs.currTickPtr, p.procInternals)
		gs.idealDC.addProc(procCopy)

		machineToUse, coreToUse := gs.pickMachine(p)

		// place proc on chosen machine
		machineToUse.sched.placeProc(p, coreToUse)
		toWrite = fmt.Sprintf("%v, %v, %v, %v, %v\n", int(*gs.currTickPtr), machineToUse.mid, p.procInternals.procType, float64(p.priority), float64(p.procInternals.actualComp))
		logWrite(ADDED_PROCS, toWrite)
		p = gs.getProc()
	}

}

// TODO this

func (gs *GlobalSched) pickMachine(procToPlace *Proc) (*Machine, Tid) {

	var machineToUse *Machine
	var coreToUse Tid
	minWaitTime := Tftick(math.MaxFloat64)
	machinesToTry := pickRandomElements(Values(gs.machines), gs.k_choices)

	for _, m := range machinesToTry {
		waitTime, coreId := m.sched.tryPlace(procToPlace)
		if waitTime < minWaitTime {
			machineToUse = m
			coreToUse = coreId
		}
	}

	return machineToUse, coreToUse
}

func (gs *GlobalSched) getProc() *Proc {
	return gs.procq.deq()
}

func (gs *GlobalSched) putProc(proc *Proc) {
	gs.procq.enq(proc)
}
