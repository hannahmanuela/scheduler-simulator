package slasched

import (
	"fmt"
	"math"
)

type TIdleMachine struct {
	machine            Tid
	highestCostRunning float32
	qlen               int
	memAvail           Tmem
}

type IdleHeap []TIdleMachine

func (h IdleHeap) Len() int           { return len(h) }
func (h IdleHeap) Less(i, j int) bool { return h[i].memAvail < h[j].memAvail }
func (h IdleHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *IdleHeap) Push(x any)        { *h = append(*h, x.(TIdleMachine)) }

func (h *IdleHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func findMostIdle(h *IdleHeap) (TIdleMachine, bool) {

	maxMemAvail := Tmem(0)
	// minQlen := math.MaxInt
	indToUse := -1

	for ind := 0; ind < len(*h); ind++ {
		idleMachine := (*h)[ind]
		// trade memory and qlen off?
		// if idleMachine.qlen < minQlen {
		// indToUse = ind
		// minQlen = idleMachine.qlen
		// }
		if idleMachine.memAvail > maxMemAvail {
			maxMemAvail = idleMachine.memAvail
			indToUse = ind
		}
	}

	if indToUse < 0 {
		return TIdleMachine{}, false
	} else {
		toRet := (*h)[indToUse]
		return toRet, true
	}

}

type MineGSS struct {
	gsId            Tid
	machines        map[Tid]*Machine
	idleMachines    *IdleHeap
	multiq          MultiQueue
	currTickPtr     *Tftick
	nProcGenPerTick int
	nFoundIdle      int
	nUsedLowLoad    int
	nUsedHighLoad   int
	nUsedKChoices   int
}

func newMineGSS(id int, machines map[Tid]*Machine, currTickPtr *Tftick, numGenPerTick int, idleHeap *IdleHeap) *MineGSS {
	gs := &MineGSS{
		gsId:            Tid(id),
		machines:        machines,
		idleMachines:    idleHeap,
		multiq:          NewMultiQ(),
		currTickPtr:     currTickPtr,
		nProcGenPerTick: numGenPerTick,
		nFoundIdle:      0,
		nUsedLowLoad:    0,
		nUsedHighLoad:   0,
		nUsedKChoices:   0,
	}

	return gs
}

func (gs *MineGSS) MachinesString() string {
	str := "machines: \n"
	for _, m := range gs.machines {
		str += "   " + m.String()
	}
	return str
}

func (gs *MineGSS) placeProcs() {

	logWrite(SCHED, "\n")

	toWrite := fmt.Sprintf("idle list before placing procs: %v \n", gs.idleMachines)
	logWrite(SCHED, toWrite)

	// setup
	p := gs.multiq.deq(*gs.currTickPtr)

	toReq := make([]*Proc, 0)

	for p != nil {
		// place given proc

		machineToUse := gs.pickMachine()

		toWrite := fmt.Sprintf("%v, GS %v placing proc %v; new idle heap: %v\n", int(*gs.currTickPtr), gs.gsId, p.procId, gs.idleMachines)
		logWrite(SCHED, toWrite)

		if machineToUse == nil {
			logWrite(SCHED, "    -> nothing avail \n")
			toReq = append(toReq, p)
			p = gs.multiq.deq(*gs.currTickPtr)
			continue
		}

		machineToUse.placeProc(p)
		toWrite = fmt.Sprintf("    -> chose %v\n", machineToUse.machineId)
		logWrite(SCHED, toWrite)

		p = gs.multiq.deq(*gs.currTickPtr)
	}

	for _, p := range toReq {
		gs.multiq.enq(p)
	}

	if *gs.currTickPtr == 199 {
		fmt.Printf("num used idle: %v, num k choices: %v (low: %v, high: %v)\n", gs.nFoundIdle, gs.nUsedKChoices, gs.nUsedLowLoad, gs.nUsedHighLoad)
	}

}

func (gs *MineGSS) pickMachine() *Machine {

	machine, found := findMostIdle(gs.idleMachines)
	if found {
		gs.nFoundIdle += 1
		// begNumInList := gs.idleMachines.Len()
		remove(gs.idleMachines, machine.machine)
		machine.memAvail -= MAX_MEM / 2
		machine.qlen += 1
		if machine.qlen < MIN_QLEN_IDLE_LIST {
			// fmt.Printf("%v idle q size b4 %v after%v, machine %v new mem avail %v qlen %v \n", *gs.currTickPtr, begNumInList, gs.idleMachines.Len(), machine.machine, machine.memAvail, machine.qlen)
			gs.idleMachines.Push(machine)
		}
		return gs.machines[machine.machine]
	}

	gs.nUsedKChoices += 1

	// if no idle machine, use power of k choices
	var machineToUse *Machine
	machineToTry := pickRandomElements(Values(gs.machines), K_CHOICES_DOWN)

	minQlen := math.MaxInt

	for _, m := range machineToTry {

		if m.procQ.qlen() < minQlen {
			minQlen = m.procQ.qlen()
			machineToUse = m
		}
	}

	return machineToUse

	// minMemPaged := Tmem(math.MaxInt)
	// var machineMinMemPaged *Machine
	// minProcsJustStartedHigh := math.MaxInt
	// var machineMinProcsJustStartedHigh *Machine

	// // low load setting - for machines with no mem paged, we want to look at memFree combined with number of just started procs
	// maxMemFree := Tmem(0)
	// var machineMaxMemFree *Machine
	// minProcsJustStartedLow := math.MaxInt
	// var machineMinProcsJustStartedLow *Machine

	// // things we could get from machine:
	// // - qlen
	// // - number of just started procs
	// // - mem free
	// // - mem paged

	// // - if all have mem paged, choose the one with the least mem paged (and new procs - within buffer)
	// for _, m := range machineToTry {

	// 	if m.memPaged() == 0 {
	// 		if m.memFree() > maxMemFree {
	// 			maxMemFree = m.memFree()
	// 			machineMaxMemFree = m
	// 		}
	// 		if m.numProcsJustStarted() < minProcsJustStartedLow {
	// 			minProcsJustStartedLow = m.numProcsJustStarted()
	// 			machineMinProcsJustStartedLow = m
	// 		}
	// 		continue
	// 	}

	// 	if m.memPaged() < minMemPaged {
	// 		minMemPaged = m.memPaged()
	// 		machineMinMemPaged = m
	// 	}

	// 	if m.numProcsJustStarted() < minProcsJustStartedHigh {
	// 		minProcsJustStartedHigh = m.numProcsJustStarted()
	// 		machineMinProcsJustStartedHigh = m
	// 	}
	// }

	// if machineMaxMemFree != nil {
	// 	gs.nUsedLowLoad += 1
	// 	// low load setting - some machines have nothing paged out -- this could also be (another) idle list?
	// 	diffInMemFree := machineMaxMemFree.memFree() - machineMinProcsJustStartedLow.memFree()
	// 	diffInProcsStarted := machineMinProcsJustStartedLow.numProcsJustStarted() - machineMaxMemFree.numProcsJustStarted()

	// 	if int(diffInMemFree) > 5000*diffInProcsStarted {
	// 		machineToUse = machineMaxMemFree
	// 	} else {
	// 		machineToUse = machineMinProcsJustStartedLow
	// 	}
	// } else {
	// 	gs.nUsedHighLoad += 1
	// 	// high load setting - all the machines have something paged out
	// 	diffInMemPaged := machineMinProcsJustStartedHigh.memPaged() - machineMinMemPaged.memPaged()
	// 	diffInProcsStarted := machineMinMemPaged.numProcsJustStarted() - machineMinProcsJustStartedHigh.numProcsJustStarted()

	// 	if int(diffInMemPaged) > 5000*diffInProcsStarted {
	// 		machineToUse = machineMinMemPaged
	// 	} else {
	// 		machineToUse = machineMinProcsJustStartedHigh
	// 	}
	// }

	// return machineToUse
}
