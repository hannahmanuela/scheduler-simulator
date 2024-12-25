package slasched

import (
	"fmt"
	"math"
)

const (
	TICK_SCHED_THRESHOLD = 0.001 // amount of ticks after which I stop scheduling; given that 1 tick = 5ms (see website.go)
)

type Sched struct {
	machineId               Tid
	numCores                int
	activeQ                 *Queue
	idleHeap                *IdleHeap
	currTickPtr             *Tftick
	worldNumProcsGenPerTick int
}

func newSched(numCores int, idleHeap *IdleHeap, mid Tid, currTickPtr *Tftick, nGenPerTick int) *Sched {
	sd := &Sched{
		machineId:               mid,
		numCores:                numCores,
		activeQ:                 newQueue(),
		idleHeap:                idleHeap,
		currTickPtr:             currTickPtr,
		worldNumProcsGenPerTick: nGenPerTick,
	}
	return sd
}

func (sd *Sched) String() string {
	return fmt.Sprintf("machine scheduler: %v", sd.machineId)
}

func (sd *Sched) tick() {
	sd.simulateRunProcs()
}

func (sd *Sched) memFree() Tmem {

	memUsed := Tmem(0)

	for _, p := range sd.activeQ.getQ() {
		memUsed += p.maxMem()
	}
	return MEM_PER_MACHINE - memUsed
}

func (sd *Sched) okToPlace(newProc *Proc) float32 {

	// if it just fits in terms of memory do it
	if newProc.maxMem() < sd.memFree() {
		return 0
	}

	// if it doesn't fit, look if there a good proc to kill? (/a combination of procs? can add that later)
	_, minMoneyWaste := sd.activeQ.checkKill(newProc)

	return minMoneyWaste
}

func (sd *Sched) placeProc(newProc *Proc) {

	newProc.timePlaced = *sd.currTickPtr

	if newProc.maxMem() < sd.memFree() {
		sd.activeQ.enq(newProc)

		toWrite := fmt.Sprintf("placing pid %v, ok b/c mem fits \n", newProc.procId)
		logWrite(SCHED, toWrite)

		return
	}

	// if it doesn't fit, look if there a good proc to kill? (/a combination of procs? can add that later)
	procToKill, _ := sd.activeQ.checkKill(newProc)

	toWrite := fmt.Sprintf("killing pid %v to place pid %v \n", procToKill, newProc.procId)
	logWrite(SCHED, toWrite)

	sd.activeQ.kill(procToKill)
	sd.activeQ.enq(newProc)
}

// do numCores ticks of computation (only on procs in the activeQ)
func (sd *Sched) simulateRunProcs() {

	totalTicksLeftToGive := Tftick(sd.numCores)
	ticksLeftPerCore := make(map[int]Tftick, 0)
	coresWithTicksLeft := make(map[int]bool, 0)
	coresLeftThisRound := make(map[int]bool, 0)

	for i := 0; i < sd.numCores; i++ {
		ticksLeftPerCore[i] = Tftick(1)
		coresWithTicksLeft[i] = true
	}

	toWrite := fmt.Sprintf("%v, %v, %v", sd.worldNumProcsGenPerTick, int(*sd.currTickPtr), sd.machineId)
	logWrite(USAGE, toWrite)

	putProcOnCoreWithMaxTimeLeft := func() int {
		minVal := Tftick(math.MaxFloat32)
		minCore := -1
		for i := 0; i < sd.numCores; i++ {
			if _, ok := coresLeftThisRound[i]; ok {
				if _, ok := coresWithTicksLeft[i]; ok {
					if ticksLeftPerCore[i] < minVal {
						minVal = ticksLeftPerCore[i]
						minCore = i
					}
				}
			}
		}
		delete(coresLeftThisRound, minCore)
		return minCore
	}

	toReq := make([]*Proc, 0)

	toWrite = fmt.Sprintf("%v @ %v, machine %v; has q %v\n", sd.worldNumProcsGenPerTick, sd.currTickPtr.String(), sd.machineId, sd.activeQ.String())
	logWrite(SCHED, toWrite)

	for sd.activeQ.qlen() > 0 && totalTicksLeftToGive-Tftick(TICK_SCHED_THRESHOLD) > 0.0 {

		for i := 0; i < sd.numCores; i++ {
			coresLeftThisRound[i] = true
		}

		// run by amount of money willing to spend
		toWrite := fmt.Sprintf("  q len before %v \n", sd.activeQ.qlen())
		logWrite(SCHED, toWrite)
		coreToProc := make(map[int]*Proc, 0)
		for i := 0; i < sd.numCores; i++ {
			p := sd.activeQ.deq()
			if p == nil {
				continue
			}
			coreToUse := putProcOnCoreWithMaxTimeLeft()
			if coreToUse == -1 {
				sd.activeQ.enq(p)
				coreToProc[coreToUse] = nil
			} else {
				coreToProc[coreToUse] = p
			}
		}
		toWrite = fmt.Sprintf("  q len after %v; assignment: %v \n", sd.activeQ.qlen(), coreToProc)
		logWrite(SCHED, toWrite)

		// run all the cores once
		for currCore := 0; currCore < sd.numCores; currCore++ {

			procToRun := coreToProc[currCore]

			if procToRun == nil {
				continue
			}

			toWrite := fmt.Sprintf("   core %v giving %v to proc %v \n", currCore, ticksLeftPerCore[currCore], procToRun.String())
			logWrite(SCHED, toWrite)

			ticksUsed, done := procToRun.runTillOutOrDone(ticksLeftPerCore[currCore])

			ticksLeftPerCore[currCore] -= ticksUsed
			totalTicksLeftToGive -= ticksUsed

			if ticksLeftPerCore[currCore] < TICK_SCHED_THRESHOLD {
				delete(coresWithTicksLeft, currCore)
			}

			if !done {
				toReq = append(toReq, procToRun)
			} else {
				// if the proc is done, update the ticksPassed to be exact for metrics etc
				procToRun.timeDone = *sd.currTickPtr + (1 - ticksLeftPerCore[currCore])

				toWrite := fmt.Sprintf("   -> done: %v\n", procToRun.String())
				logWrite(SCHED, toWrite)

				toWrite = fmt.Sprintf("%v, %v, %v, %v \n", sd.worldNumProcsGenPerTick, procToRun.willingToSpend(), (procToRun.timeDone - procToRun.timeStarted).String(), procToRun.compDone.String())
				logWrite(PROCS_DONE, toWrite)
			}
		}

	}

	for _, p := range toReq {
		sd.activeQ.enq(p)
	}

	if totalTicksLeftToGive < 0.00002 {
		totalTicksLeftToGive = 0
	}
	toWrite = fmt.Sprintf(", %v, %v\n", float64(math.Copysign(float64(totalTicksLeftToGive), 1)), sd.memFree())
	logWrite(USAGE, toWrite)

	highestCost := float32(0)
	for _, p := range sd.activeQ.getQ() {
		if p.willingToSpend() > highestCost {
			highestCost = p.willingToSpend()
		}
	}

	sd.idleHeap.lock.Lock()
	// also if it is already in the heap, then replace it with the new value
	if contains(sd.idleHeap.heap, sd.machineId) {
		remove(sd.idleHeap.heap, sd.machineId)
	}
	toPush := TIdleMachine{
		memAvail:           sd.memFree(),
		highestCostRunning: highestCost,
		machine:            sd.machineId,
	}
	sd.idleHeap.heap.Push(toPush)
	sd.idleHeap.lock.Unlock()

}
