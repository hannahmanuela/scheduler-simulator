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

func (sd *Sched) printAllProcs() {

	for _, p := range sd.activeQ.getQ() {
		toWrite := fmt.Sprintf("%v, %v, %v, %v, %v\n", int(*sd.currTickPtr), sd.machineId, p.procId, float64(p.procInternals.actualComp), float64(p.compDone))
		logWrite(CURR_PROCS, toWrite)
	}
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
	sd.activeQ.enq(newProc)
}

// do numCores ticks of computation (only on procs in the activeQ)
func (sd *Sched) simulateRunProcs() {

	totalTicksLeftToGive := Tftick(sd.numCores)
	ticksLeftPerCore := make(map[int]Tftick, 0)
	coresLeft := make(map[int]bool, 0)

	for i := 0; i < sd.numCores; i++ {
		ticksLeftPerCore[i] = Tftick(1)
		coresLeft[i] = true
	}

	toWrite := fmt.Sprintf("%v, %v, %v, %v", sd.worldNumProcsGenPerTick, int(*sd.currTickPtr), -1, sd.activeQ.qlen())
	logWrite(IDEAL_USAGE, toWrite)

	putProcOnCoreWithMaxTimeLeft := func() int {
		minVal := Tftick(math.MaxFloat32)
		minCore := -1
		for i := 0; i < sd.numCores; i++ {
			if _, ok := coresLeft[i]; ok {
				if ticksLeftPerCore[i] < minVal {
					minVal = ticksLeftPerCore[i]
					minCore = i
				}
			}
		}
		delete(coresLeft, minCore)
		return minCore
	}

	toReq := make([]*Proc, 0)

	for sd.activeQ.qlen() > 0 && totalTicksLeftToGive-Tftick(TICK_SCHED_THRESHOLD) > 0.0 {

		// run by amount of money willing to spend
		coreToProc := make(map[int]*Proc, 0)
		for i := 0; i < sd.numCores; i++ {
			p := sd.activeQ.deq()
			if p == nil {
				continue
			}
			coreToUse := putProcOnCoreWithMaxTimeLeft()
			coreToProc[coreToUse] = p
		}

		// run all the cores once
		for currCore := 0; currCore < sd.numCores; currCore++ {

			procToRun := coreToProc[currCore]

			if procToRun == nil {
				continue
			}

			toWrite := fmt.Sprintf("   giving %v to proc %v\n", ticksLeftPerCore[currCore], procToRun.String())
			logWrite(IDEAL_SCHED, toWrite)

			ticksUsed, done := procToRun.runTillOutOrDone(ticksLeftPerCore[currCore])

			ticksLeftPerCore[currCore] -= ticksUsed
			totalTicksLeftToGive -= ticksUsed

			if !done {
				toReq = append(toReq, procToRun)
			} else {
				// if the proc is done, update the ticksPassed to be exact for metrics etc
				procToRun.timeDone = *sd.currTickPtr + (1 - ticksLeftPerCore[currCore])
			}

		}

	}

	for _, p := range toReq {
		sd.activeQ.enq(p)
	}

	if totalTicksLeftToGive < 0.00002 {
		totalTicksLeftToGive = 0
	}
	toWrite = fmt.Sprintf(", %v\n", float64(math.Copysign(float64(totalTicksLeftToGive), 1)))
	logWrite(USAGE, toWrite)

	sd.idleHeap.lock.Lock()
	// also if it is already in the heap, then replace it with the new value
	if contains(sd.idleHeap.heap, sd.machineId) {
		remove(sd.idleHeap.heap, sd.machineId)
	}
	toPush := TIdleMachine{
		memAvail: sd.memFree(),
		machine:  sd.machineId,
	}
	sd.idleHeap.heap.Push(toPush)
	sd.idleHeap.lock.Unlock()

}
