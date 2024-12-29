package slasched

import (
	"fmt"
	"math"
	"sync"
)

type TIdleMachine struct {
	machine            Tid
	highestCostRunning float32
	qlen               int
	memAvail           Tmem
}

// TODO: basically we can think of this as a free list, should treat it accordingly (this is a well-known problem)
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

func useBestIdle(h *MinHeap, memNeeded Tmem) (TIdleMachine, bool) {

	// under mem pressures: choose based off memory fitting

	// if there are many where it would fit, pick based off qlen and highestCostRunning

	possMachines := make([]TIdleMachine, 0)

	for ind := 0; ind < len(*h); ind++ {
		idleMachine := (*h)[ind]
		if idleMachine.memAvail > memNeeded {
			possMachines = append(possMachines, idleMachine)
		}
	}

	// minHighestCost := float32(math.MaxFloat32)
	minQlen := math.MaxInt
	indToUse := -1

	for ind, idleMachine := range possMachines {
		// trade qlen and priority off?
		if idleMachine.qlen < minQlen {
			indToUse = ind
			minQlen = idleMachine.qlen
		}
		// if idleMachine.qlen < minQlen && idleMachine.highestCostRunning < minHighestCost {
		// 	indToUse = ind
		// 	minQlen = idleMachine.qlen
		// 	minHighestCost = idleMachine.highestCostRunning
		// } else if idleMachine.qlen < minQlen && idleMachine.highestCostRunning > minHighestCost {
		// 	// current is running more expensive procs but has a shorter q
		// 	diffInQlen := minQlen-idleMachine.qlen
		// 	diffInCostRunning := idleMachine.highestCostRunning - minHighestCost
		// 	if diffInQlen > diffInCostRunning {
		// 		indToUse = ind
		// 		minQlen = idleMachine.qlen
		// 		minHighestCost = idleMachine.highestCostRunning
		// 	}
		// } else if idleMachine.qlen > minQlen && idleMachine.highestCostRunning < minHighestCost {
		// 	// current is running a less expensive proc but has a longer q
		// }
	}

	if indToUse < 0 {
		return TIdleMachine{}, false
	} else {
		toRet := possMachines[indToUse]
		return toRet, true
	}

}

type IdleHeap struct {
	heap *MinHeap
	lock sync.RWMutex
}

type GlobalSched struct {
	gsId            Tid
	machines        map[Tid]*Machine
	idleMachines    *IdleHeap
	multiq          MultiQueue
	currTickPtr     *Tftick
	nProcGenPerTick int
	nFoundIdle      int
	nUsedKChoices   int
}

func newGolbalSched(id int, machines map[Tid]*Machine, currTickPtr *Tftick, numGenPerTick int, idleHeap *IdleHeap) *GlobalSched {
	gs := &GlobalSched{
		gsId:            Tid(id),
		machines:        machines,
		idleMachines:    idleHeap,
		multiq:          NewMultiQ(),
		currTickPtr:     currTickPtr,
		nProcGenPerTick: numGenPerTick,
		nFoundIdle:      0,
		nUsedKChoices:   0,
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

	// toWrite := fmt.Sprintf("%v, %v: q before placing procs: %v \n", *gs.currTickPtr, gs.gsId, gs.multiq.qMap)
	// logWrite(SCHED, toWrite)

	logWrite(SCHED, "\n")

	// setup
	p := gs.multiq.deq(*gs.currTickPtr)

	toReq := make([]*Proc, 0)

	for p != nil {
		// place given proc

		machineToUse := gs.pickMachine(p)

		toWrite := fmt.Sprintf("%v, GS %v placing proc %v; curr idle heap: %v \n", int(*gs.currTickPtr), gs.gsId, p.procId, gs.idleMachines.heap)
		logWrite(SCHED, toWrite)

		if machineToUse == nil {
			logWrite(SCHED, "    -> nothing avail \n")
			toReq = append(toReq, p)
			p = gs.multiq.deq(*gs.currTickPtr)
			continue
		}

		shouldStoreIdleInfo, idleVal := machineToUse.sched.placeProc(p, gs.gsId)
		toWrite = fmt.Sprintf("    -> chose %v; after placing should store: %v, new idle val: %v \n", machineToUse.mid, shouldStoreIdleInfo, idleVal)
		logWrite(SCHED, toWrite)

		if shouldStoreIdleInfo {
			if contains(gs.idleMachines.heap, machineToUse.sched.machineId) {
				remove(gs.idleMachines.heap, machineToUse.sched.machineId)
			}
			if idleVal.memAvail > IDLE_HEAP_MEM_THRESHOLD {
				gs.idleMachines.heap.Push(idleVal)
			}
		}

		p = gs.multiq.deq(*gs.currTickPtr)
	}

	for _, p := range toReq {
		gs.multiq.enq(p)
	}

}

func (gs *GlobalSched) pickMachine(procToPlace *Proc) *Machine {

	gs.idleMachines.lock.Lock()
	machine, found := useBestIdle(gs.idleMachines.heap, procToPlace.maxMem())
	gs.idleMachines.lock.Unlock()
	if found {
		gs.nFoundIdle += 1
		return gs.machines[machine.machine]
	}

	// actualMemFree := make([]Tmem, len(gs.machines))
	// for i, m := range gs.machines {
	// 	actualMemFree[i] = m.sched.memFree()
	// }
	// fmt.Printf("%v found no good machine: memNeeded %v idle heap: %v, actual mems free: %v \n", *gs.currTickPtr, procToPlace.maxMem(), gs.idleMachines.heap, actualMemFree)

	gs.nUsedKChoices += 1

	// if no idle machine, use power of k choices (for now k = number of machines :D )
	var machineToUse *Machine
	machineToTry := pickRandomElements(Values(gs.machines), K_CHOICES_DOWN)

	minMoneyWaste := float32(math.MaxFloat32)

	for _, m := range machineToTry {
		moneyWaste := m.sched.okToPlace(procToPlace)
		toWrite := fmt.Sprintf("  min money waste: %v \n", moneyWaste)
		logWrite(SCHED, toWrite)
		if moneyWaste < minMoneyWaste {
			minMoneyWaste = moneyWaste
			machineToUse = m
		}
	}

	if minMoneyWaste > MONEY_WASTE_THRESHOLD {
		return nil
	}

	// toWrite = fmt.Sprintf("   used k choices: the machine to use is %v \n", machineToUse)
	// logWrite(SCHED, toWrite)

	return machineToUse
}
