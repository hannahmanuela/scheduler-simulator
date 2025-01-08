package slasched

import (
	"fmt"
	"math"
)

const (
	MEM_FUDGE_FACTOR_POLLING = 1.1
)

type Tid int

type Ttickmap map[Tid]Tftick
type Tprocmap map[Tid]int

type Machine struct {
	machineId               Tid
	numCores                int
	procQ                   *Queue
	idleHeaps               map[Tid]*IdleHeap
	currHeapGSS             Tid
	currTickPtr             *Tftick
	worldNumProcsGenPerTick int
}

func newMachine(mid Tid, idleHeaps map[Tid]*IdleHeap, numCores int, currTickPtr *Tftick, nGenPerTick int) *Machine {

	sd := &Machine{
		machineId:               mid,
		numCores:                numCores,
		procQ:                   newQueue(),
		idleHeaps:               idleHeaps,
		currHeapGSS:             -1,
		currTickPtr:             currTickPtr,
		worldNumProcsGenPerTick: nGenPerTick,
	}

	// add machine to an idle heap
	heapsToLookAt := pickRandomElementsMap(sd.idleHeaps, K_CHOICES_UP)

	var gsHeapToUse Tid
	minLength := math.MaxInt

	for gsId, possHeap := range heapsToLookAt {
		if possHeap.Len() < minLength {
			minLength = possHeap.Len()
			gsHeapToUse = gsId
		}
	}

	sd.currHeapGSS = gsHeapToUse
	heapToUse := sd.idleHeaps[gsHeapToUse]
	toPush := TIdleMachine{
		memAvail:           MEM_PER_MACHINE,
		highestCostRunning: -1,
		qlen:               0,
		machine:            Tid(sd.machineId),
	}
	heapToUse.Push(toPush)

	return sd
}

func (sd *Machine) String() string {
	return fmt.Sprintf("machine scheduler: %v", sd.machineId)
}

func (sd *Machine) tick() {
	sd.simulateRunProcs()
}

func (sd *Machine) memFree() Tmem {

	memUsed := Tmem(0)

	for _, p := range sd.procQ.getQ() {
		if !p.currentlyPaged {
			memUsed += p.memUsing
		}
	}
	return MEM_PER_MACHINE - memUsed
}

func (sd *Machine) memPaged() Tmem {

	memPaged := Tmem(0)

	for _, p := range sd.procQ.getQ() {
		if p.currentlyPaged {
			memPaged += p.memUsing
		}
	}
	return memPaged
}

func (sd *Machine) placeProc(newProc *Proc) {

	if sd.memFree() < 0 {
		fmt.Println("here1")
	}

	newProc.timePlaced = *sd.currTickPtr
	sd.procQ.enq(newProc)

	if sd.memFree() < 0 {
		sd.procQ.pageOut(-sd.memFree(), sd.procQ.getQ())
	}
}

func (sd *Machine) currMemFree(allProcsRunning []*Proc) Tmem {

	memUsed := Tmem(0)

	for _, p := range allProcsRunning {
		if !p.currentlyPaged {
			memUsed += p.memUsing
		}
	}

	return MEM_PER_MACHINE - memUsed
}

func (sd *Machine) simulateRunProcs() {

	allProcsRunning := make([]*Proc, sd.procQ.qlen())
	for i, p := range sd.procQ.getQ() {
		allProcsRunning[i] = p
	}

	totalTicksLeftToGive := Tftick(sd.numCores)
	ticksLeftPerCore := make(map[int]Tftick, 0)
	coresWithTicksLeft := make(map[int]bool, 0)
	coresLeftThisRound := make(map[int]bool, 0)

	for i := 0; i < sd.numCores; i++ {
		ticksLeftPerCore[i] = Tftick(1)
		coresWithTicksLeft[i] = true
	}

	toWrite := fmt.Sprintf("%v, %v, %v, %v, %v, ", sd.worldNumProcsGenPerTick, int(*sd.currTickPtr), sd.machineId, sd.memFree(), sd.memPaged())
	logWrite(USAGE, toWrite)

	putProcOnCoreWithMaxTimeLeft := func() int {
		maxVal := Tftick(0.0)
		coreToUse := -1
		for i := 0; i < sd.numCores; i++ {
			if _, ok := coresLeftThisRound[i]; ok {
				if _, ok := coresWithTicksLeft[i]; ok {
					if ticksLeftPerCore[i] > maxVal {
						maxVal = ticksLeftPerCore[i]
						coreToUse = i
					}
				}
			}
		}
		delete(coresLeftThisRound, coreToUse)
		return coreToUse
	}

	toReq := make([]*Proc, 0)

	toWrite = fmt.Sprintf("\n==> %v @ %v, machine %v (on heap: %v, mem free: %v); has q: \n%v", sd.worldNumProcsGenPerTick, sd.currTickPtr.String(), sd.machineId, sd.currHeapGSS, sd.memFree(), sd.procQ.SummaryString())
	logWrite(SCHED, toWrite)

	for sd.procQ.qlen() > 0 && totalTicksLeftToGive-Tftick(TICK_SCHED_THRESHOLD) > 0.0 && len(coresWithTicksLeft) > 0 {

		for i := 0; i < sd.numCores; i++ {
			coresLeftThisRound[i] = true
		}

		// run by amount of money willing to spend
		coreToProc := make(map[int]*Proc, 0)
		for i := 0; i < sd.numCores; i++ {
			p := sd.procQ.deq()
			if p == nil {
				continue
			}
			coreToUse := putProcOnCoreWithMaxTimeLeft()
			if coreToUse == -1 {
				sd.procQ.enq(p)
				coreToProc[coreToUse] = nil
			} else {
				coreToProc[coreToUse] = p
			}
		}

		// run all the cores once
		for currCore := 0; currCore < sd.numCores; currCore++ {

			procToRun := coreToProc[currCore]

			if procToRun == nil {
				continue
			}

			if procToRun.currentlyPaged {
				fmt.Printf("running paged proc\n")
			}

			toWrite := fmt.Sprintf("   core %v giving %v to proc %v, ", currCore, ticksLeftPerCore[currCore], procToRun.String())
			logWrite(SCHED, toWrite)

			memUseDelta, ticksUsed, done := procToRun.runTillOutOrDone(ticksLeftPerCore[currCore])

			ticksLeftPerCore[currCore] -= ticksUsed
			totalTicksLeftToGive -= ticksUsed

			toWrite = fmt.Sprintf("mem use delta: %v, new mem free: %v \n", memUseDelta, sd.currMemFree(allProcsRunning))
			logWrite(SCHED, toWrite)

			if ticksLeftPerCore[currCore] < TICK_SCHED_THRESHOLD {
				delete(coresWithTicksLeft, currCore)
			}

			if !done {
				toReq = append(toReq, procToRun)

				if sd.currMemFree(allProcsRunning) < 0 {
					toWrite = fmt.Sprintf("      mem usg over: %v, pagingOUT\n", sd.currMemFree(allProcsRunning))
					logWrite(SCHED, toWrite)
					sd.procQ.pageOut(-sd.currMemFree(allProcsRunning), allProcsRunning)
					toWrite = fmt.Sprintf("      - new curr mem free: %v\n", sd.currMemFree(allProcsRunning))
					logWrite(SCHED, toWrite)
				}

			} else {
				// if the proc is done, update the ticksPassed to be exact for metrics etc
				procToRun.timeDone = *sd.currTickPtr + (1 - ticksLeftPerCore[currCore])

				toWrite := fmt.Sprintf("   -> done: %v\n", procToRun.String())
				logWrite(SCHED, toWrite)

				allProcsRunning = removeFromList(allProcsRunning, procToRun)

				if sd.memPaged() > 0 {
					sd.procQ.pageIn(sd.currMemFree(allProcsRunning))
					toWrite := fmt.Sprintf("    pagedIN: new mem free: %v\n", sd.currMemFree(allProcsRunning))
					logWrite(SCHED, toWrite)
				}

				if (procToRun.timeDone - procToRun.timeStarted) > procToRun.compDone {
					toWrite := fmt.Sprintf("   ---> OVER %v \n", procToRun.String())
					logWrite(SCHED, toWrite)
				}

				toWrite = fmt.Sprintf("%v, %v, %v, %v \n", sd.worldNumProcsGenPerTick, procToRun.willingToSpend(), (procToRun.timeDone - procToRun.timeStarted).String(), procToRun.compDone.String())
				logWrite(PROCS_DONE, toWrite)
			}
		}

	}

	for _, p := range toReq {
		sd.procQ.enq(p)
	}

	toWrite = fmt.Sprintf("q at end: %v \n\n", sd.procQ.String())
	logWrite(SCHED, toWrite)

	if totalTicksLeftToGive < 0.00002 {
		totalTicksLeftToGive = 0
	}
	toWrite = fmt.Sprintf("%.3f, %v, %v\n", float64(math.Copysign(float64(totalTicksLeftToGive), 1)), sd.memFree(), sd.memPaged())
	logWrite(USAGE, toWrite)

	highestCost := float32(0)
	for _, p := range sd.procQ.getQ() {
		if p.willingToSpend() > highestCost {
			highestCost = p.willingToSpend()
		}
	}

	if (sd.memPaged() > 0) || (sd.memFree() < INIT_MEM) {
		// not idle
		return
	}

	// only here if we are idle
	var heapToUse *IdleHeap
	if sd.currHeapGSS >= 0 {
		// already in a heap, need to just update that info
		heapToUse = sd.idleHeaps[sd.currHeapGSS]
	} else {
		// choose idle heap to use by power of k choices
		heapsToLookAt := pickRandomElementsMap(sd.idleHeaps, K_CHOICES_UP)

		minLength := math.MaxInt
		for gssId, possHeap := range heapsToLookAt {
			if possHeap.Len() < minLength {
				minLength = possHeap.Len()
				heapToUse = possHeap
				sd.currHeapGSS = gssId
			}
		}
	}

	// also if it is already in the heap, then replace it with the new value
	if contains(heapToUse, sd.machineId) {
		remove(heapToUse, sd.machineId)
	}
	toPush := TIdleMachine{
		machine:            sd.machineId,
		highestCostRunning: highestCost,
		qlen:               sd.procQ.qlen(),
		memAvail:           sd.memFree(),
	}
	heapToUse.Push(toPush)

}
