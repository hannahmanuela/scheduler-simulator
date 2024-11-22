package slasched

import (
	"container/heap"
	"fmt"
	"sync"
)

type TmachineCoreId struct {
	machineId Tid
	coreId    Tid
}

type TIdleMachine struct {
	compIdleFor   Tftick
	machineCoreId TmachineCoreId
}
type MinHeap []TIdleMachine

func (h MinHeap) Len() int           { return len(h) }
func (h MinHeap) Less(i, j int) bool { return h[i].compIdleFor < h[j].compIdleFor }
func (h MinHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *MinHeap) Push(x any)        { *h = append(*h, x.(TIdleMachine)) }

func (h *MinHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func popNextLarger(h *MinHeap, value Tftick) (TIdleMachine, bool) {
	var tempHeap MinHeap

	for h.Len() > 0 {
		item := heap.Pop(h).(TIdleMachine)

		if item.compIdleFor > value {
			return item, true
		}
		tempHeap = append(tempHeap, item)
	}

	for _, item := range tempHeap {
		heap.Push(h, item)
	}
	return TIdleMachine{}, false
}

type IdleHeap struct {
	heap *MinHeap
	lock sync.RWMutex
}

type GlobalSched struct {
	machines        map[Tid]*Machine
	k_choices       int
	idleMachines    *IdleHeap
	procq           *Queue
	currTickPtr     *Tftick
	numFoundIdle    map[int]int
	numUsedKChoices map[int]int
	numSaidNo       map[int]int
	numStarted      map[int]int
}

func newGolbalSched(machines map[Tid]*Machine, currTickPtr *Tftick, idleHeap *IdleHeap) *GlobalSched {
	gs := &GlobalSched{
		machines:        machines,
		k_choices:       int(len(machines) / 3),
		idleMachines:    idleHeap,
		procq:           newQueue(),
		currTickPtr:     currTickPtr,
		numFoundIdle:    make(map[int]int),
		numUsedKChoices: make(map[int]int),
		numSaidNo:       make(map[int]int),
		numStarted:      make(map[int]int),
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

		machineToUse, coreToUse := gs.pickMachine(p)

		if machineToUse == nil {
			p = gs.getProc()
			continue
		}

		// place proc on chosen machine
		p.machineId = machineToUse.mid
		machineToUse.sched.placeProc(p, coreToUse)
		if VERBOSE_GS_STATS {
			toWrite := fmt.Sprintf("%v, %v, %v, %v, %v\n", int(*gs.currTickPtr), machineToUse.mid, p.procInternals.procType, float64(p.procInternals.deadline), float64(p.procInternals.actualComp))
			logWrite(ADDED_PROCS, toWrite)
		}
		p = gs.getProc()
	}

}

// admission control:
// 1. first look for machines that simply currently have the space (using interval tree of immediately available compute)
// 2. if there are none, do the ok to place call on all machines? on some machines? just random would be the closest to strictly following what they do...
func (gs *GlobalSched) pickMachine(procToPlace *Proc) (*Machine, Tid) {

	if _, ok := gs.numStarted[int(procToPlace.deadline)]; ok {
		gs.numStarted[int(procToPlace.deadline)] += 1
	} else {
		gs.numStarted[int(procToPlace.deadline)] = 1
	}

	gs.idleMachines.lock.Lock()
	machine, found := popNextLarger(gs.idleMachines.heap, procToPlace.maxComp)
	gs.idleMachines.lock.Unlock()
	if found {
		toWrite := fmt.Sprintf("%v, GS placing proc: %v, found an idle machine %d with %.2f comp avail \n", int(*gs.currTickPtr), procToPlace.String(), machine.machineCoreId, float64(machine.compIdleFor))
		logWrite(SCHED, toWrite)

		if _, ok := gs.numFoundIdle[int(procToPlace.deadline)]; ok {
			gs.numFoundIdle[int(procToPlace.deadline)] += 1
		} else {
			gs.numFoundIdle[int(procToPlace.deadline)] = 1
		}

		return gs.machines[machine.machineCoreId.machineId], machine.machineCoreId.coreId
	}

	if _, ok := gs.numUsedKChoices[int(procToPlace.deadline)]; ok {
		gs.numUsedKChoices[int(procToPlace.deadline)] += 1
	} else {
		gs.numUsedKChoices[int(procToPlace.deadline)] = 1
	}

	// if no idle machine, use power of k choices (for now k = number of machines :D )
	contenderMachines := make([]TmachineCoreId, 0)
	machineToTry := pickRandomElements(Values(gs.machines), gs.k_choices)

	for _, m := range machineToTry {
		if ok, coreId := m.sched.okToPlace(procToPlace); ok {
			contenderMachines = append(contenderMachines, TmachineCoreId{m.mid, coreId})
		}
	}

	toWrite := fmt.Sprintf("%v, GS placing proc: %v, the contender machines are %v \n", int(*gs.currTickPtr), procToPlace.String(), contenderMachines)
	logWrite(SCHED, toWrite)

	if len(contenderMachines) == 0 {
		toWrite := fmt.Sprintf("%v: DOESN'T FIT ANYWHERE :(( -- skipping: %v \n", int(*gs.currTickPtr), procToPlace)
		logWrite(SCHED, toWrite)
		if _, ok := gs.numSaidNo[int(procToPlace.deadline)]; ok {
			gs.numSaidNo[int(procToPlace.deadline)] += 1
		} else {
			gs.numSaidNo[int(procToPlace.deadline)] = 1
		}
		return nil, -1
	}

	// TODO: this is stupid
	machineToUse := contenderMachines[0]

	return gs.machines[machineToUse.machineId], machineToUse.coreId
}

func (gs *GlobalSched) getProc() *Proc {
	return gs.procq.deq()
}

func (gs *GlobalSched) putProc(proc *Proc) {
	gs.procq.enq(proc)
}
