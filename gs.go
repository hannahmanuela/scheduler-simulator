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
	machines     map[Tid]*Machine
	idleMachines *IdleHeap
	procq        *Queue
	currTickPtr  *Tftick
}

func newLoadBalancer(machines map[Tid]*Machine, currTickPtr *Tftick, idleHeap *IdleHeap) *GlobalSched {
	lb := &GlobalSched{
		machines:     machines,
		idleMachines: idleHeap,
		procq:        &Queue{q: make([]*Proc, 0)},
		currTickPtr:  currTickPtr,
	}

	return lb
}

func (gs *GlobalSched) MachinesString() string {
	str := "machines: \n"
	for _, m := range gs.machines {
		str += "   " + m.String()
	}
	return str
}

func (lb *GlobalSched) placeProcs() {
	// setup
	p := lb.getProc()

	toReQ := make([]*Proc, 0)

	for p != nil {
		// place given proc

		machineToUse, coreToUse := lb.pickMachine(p)

		if machineToUse == nil {
			p = lb.getProc()
			continue
		}

		// place proc on chosen machine
		p.machineId = machineToUse.mid
		machineToUse.sched.placeProc(p, coreToUse)
		if VERBOSE_LB_STATS {
			toWrite := fmt.Sprintf("%v, %v, %v, %v, %v\n", int(*lb.currTickPtr), machineToUse.mid, p.procInternals.procType, float64(p.procInternals.deadline), float64(p.procInternals.actualComp))
			logWrite(ADDED_PROCS, toWrite)
		}
		p = lb.getProc()
	}

	for _, p := range toReQ {
		lb.putProc(p)
	}
}

// admission control:
// 1. first look for machines that simply currently have the space (using interval tree of immediately available compute)
// 2. if there are none, do the ok to place call on all machines? on some machines? just random would be the closest to strictly following what they do...
func (lb *GlobalSched) pickMachine(procToPlace *Proc) (*Machine, Tid) {

	lb.idleMachines.lock.Lock()
	machine, found := popNextLarger(lb.idleMachines.heap, procToPlace.maxComp)
	lb.idleMachines.lock.Unlock()
	if found {
		toWrite := fmt.Sprintf("%v, LB placing proc: %v, found an idle machine %d with %.2f comp avail \n", int(*lb.currTickPtr), procToPlace.String(), machine.machineCoreId, float64(machine.compIdleFor))
		logWrite(SCHED, toWrite)

		return lb.machines[machine.machineCoreId.machineId], machine.machineCoreId.coreId
	}

	// if no idle machine, use power of k choices (for now k = number of machines :D )
	contenderMachines := make([]TmachineCoreId, 0)

	for _, m := range lb.machines {
		if ok, coreId := m.sched.okToPlace(procToPlace); ok {
			contenderMachines = append(contenderMachines, TmachineCoreId{m.mid, coreId})
		}
	}

	toWrite := fmt.Sprintf("%v, LB placing proc: %v, there are %v contender machines \n", int(*lb.currTickPtr), procToPlace.String(), len(contenderMachines))
	logWrite(SCHED, toWrite)

	if len(contenderMachines) == 0 {
		toWrite := fmt.Sprintf("%v: DOESN'T FIT ANYWHERE :(( -- skipping: %v \n", int(*lb.currTickPtr), procToPlace)
		logWrite(SCHED, toWrite)
		return nil, -1
	}

	// TODO: this is stupid
	machineToUse := contenderMachines[0]

	return lb.machines[machineToUse.machineId], machineToUse.coreId
}

func (lb *GlobalSched) getProc() *Proc {
	return lb.procq.deq()
}

func (lb *GlobalSched) putProc(proc *Proc) {
	lb.procq.enq(proc)
}
