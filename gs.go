package slasched

import (
	"container/heap"
	"fmt"
	"math"
	"sync"
)

type TIdleMachine struct {
	memAvail Tmem
	machine  Tid
}
type MinHeap []TIdleMachine

func (h MinHeap) Len() int           { return len(h) }
func (h MinHeap) Less(i, j int) bool { return h[i].memAvail < h[j].memAvail }
func (h MinHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *MinHeap) Push(x any)        { *h = append(*h, x.(TIdleMachine)) }

func (h *MinHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func popNextLarger(h *MinHeap, memNeeded Tmem) (TIdleMachine, bool) {
	var tempHeap MinHeap

	for h.Len() > 0 {
		item := heap.Pop(h).(TIdleMachine)

		if item.memAvail > memNeeded {
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
	idealDC         *IdealDC
	procq           *Queue
	currTickPtr     *Tftick
	nProcGenPerTick int
}

func newGolbalSched(machines map[Tid]*Machine, currTickPtr *Tftick, numGenPerTick int, idleHeap *IdleHeap, idealDC *IdealDC) *GlobalSched {
	gs := &GlobalSched{
		machines:        machines,
		k_choices:       int(len(machines) / 3),
		idleMachines:    idleHeap,
		idealDC:         idealDC,
		procq:           newQueue(),
		currTickPtr:     currTickPtr,
		nProcGenPerTick: numGenPerTick,
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

func (gs *GlobalSched) placeProcsIdeal() {
	// setup
	p := gs.getProc()

	toReq := make([]*Proc, 0)

	for p != nil {
		// place given proc

		// try placing on the ideal
		// procCopy := newProvProc(p.procId, *gs.currTickPtr, p.procInternals)
		placed := gs.idealDC.potPlaceProc(p)

		if !placed {
			toReq = append(toReq, p)
			p = gs.getProc()
			continue
		}

		p = gs.getProc()
	}

	for _, p := range toReq {
		gs.putProc(p)
	}

}

// admission control:
// 1. first look for machines that simply currently have the space (using interval tree of immediately available compute)
// 2. if there are none, do the ok to place call on all machines? on some machines? just random would be the closest to strictly following what they do...
func (gs *GlobalSched) pickMachine(procToPlace *Proc) *Machine {

	gs.idleMachines.lock.Lock()
	machine, found := popNextLarger(gs.idleMachines.heap, procToPlace.maxMem())
	gs.idleMachines.lock.Unlock()
	if found {
		return gs.machines[machine.machine]
	}

	// if no idle machine, use power of k choices (for now k = number of machines :D )
	var machineToUse *Machine
	machineToTry := pickRandomElements(Values(gs.machines), gs.k_choices)

	lowestKill := float32(math.MaxFloat32)

	for _, m := range machineToTry {
		killNeeded := m.sched.okToPlace(procToPlace)
		if killNeeded < lowestKill {
			machineToUse = m
		}
	}

	if lowestKill > MONEY_WASTE_THRESHOLD {
		return nil
	}

	toWrite := fmt.Sprintf("%v, GS placing proc: %v, the machine to use is %v \n", int(*gs.currTickPtr), procToPlace.String(), machineToUse)
	logWrite(SCHED, toWrite)

	return machineToUse
}

func (gs *GlobalSched) getProc() *Proc {
	return gs.procq.deq()
}

func (gs *GlobalSched) putProc(proc *Proc) {
	gs.procq.enq(proc)
}
