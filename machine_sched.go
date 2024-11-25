package slasched

import (
	"fmt"
	"math"
	"sort"
)

const (
	TICK_SCHED_THRESHOLD = 0.001 // amount of ticks after which I stop scheduling; given that 1 tick = 5ms (see website.go)
)

type Sched struct {
	machineId               Tid
	numCores                int
	activeQ                 map[Tid]*Queue
	idleHeap                *IdleHeap
	currTickPtr             *Tftick
	worldNumProcsGenPerTick int
}

func newSched(numCores int, idleHeap *IdleHeap, mid Tid, currTickPtr *Tftick, nGenPerTick int) *Sched {
	sd := &Sched{
		machineId:               mid,
		numCores:                numCores,
		activeQ:                 make(map[Tid]*Queue),
		idleHeap:                idleHeap,
		currTickPtr:             currTickPtr,
		worldNumProcsGenPerTick: nGenPerTick,
	}
	for i := 0; i < numCores; i++ {
		sd.activeQ[Tid(i)] = newQueue()
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

	for i := 0; i < sd.numCores; i++ {
		for _, p := range sd.activeQ[Tid(i)].getQ() {
			toWrite := fmt.Sprintf("%v, %v, 1, %v, %v, %v\n", int(*sd.currTickPtr), sd.machineId,
				float64(p.deadline), float64(p.procInternals.actualComp), float64(p.compUsed()))
			logWrite(CURR_PROCS, toWrite)
		}
	}
}

func (sd *Sched) placeProc(newProc *Proc, coreId Tid) {

	sd.activeQ[coreId].enq(newProc)

}

// checks if a proc can fit:
// a) if it has enough slack to accomodate procs with a lower deadline, and
// b) if procs with a larger deadline have enough slack to accomodate it
func (sd *Sched) okToPlace(newProc *Proc) (bool, Tid) {

	// fmt.Printf("--- running okToPlace: %v, %v \n", sd.currTick, sd.machineId)
outer:
	for currCore := 0; currCore < sd.numCores; currCore++ {

		fullList := append(make([]*Proc, 0, len(sd.activeQ[Tid(currCore)].getQ())+1), sd.activeQ[Tid(currCore)].getQ()...)
		fullList = append(fullList, newProc)
		sort.Slice(fullList, func(i, j int) bool {
			return fullList[i].deadline < fullList[j].deadline
		})

		runningWaitTime := Tftick(0)

		for _, p := range fullList {

			if float64(p.getSlack(*sd.currTickPtr)-runningWaitTime) < 0.0 {
				continue outer
			}
			runningWaitTime += p.getExpectedCompLeft()
		}

		return true, Tid(currCore)

	}

	return false, -1

}

// do numCores ticks of computation (only on procs in the activeQ)
func (sd *Sched) simulateRunProcs() {

	sum_qlens := 0
	for i := 0; i < sd.numCores; i++ {
		sum_qlens += sd.activeQ[Tid(i)].qlen()
	}

	toWrite := fmt.Sprintf("%v, %v, %v, %v", sd.worldNumProcsGenPerTick, int(*sd.currTickPtr), sd.machineId, sum_qlens)
	logWrite(USAGE, toWrite)

	ticksLeftPerCore := make(map[int]Tftick, 0)
	totalTicksLeftToGive := Tftick(sd.numCores)

	for currCore := 0; currCore < sd.numCores; currCore++ {

		ticksLeftToGive := Tftick(1)
		ticksLeftPerCore[currCore] = Tftick(1)

		toWrite := fmt.Sprintf("%v, %v, %v, curr q ACTIVE: %v \n", int(*sd.currTickPtr), sd.machineId, currCore, sd.activeQ[Tid(currCore)].String())
		logWrite(SCHED, toWrite)

		for ticksLeftToGive-Tftick(TICK_SCHED_THRESHOLD) > 0.0 && sd.activeQ[Tid(currCore)].qlen() > 0 {

			procToRun := sd.activeQ[Tid(currCore)].deq()

			if procToRun == nil {
				break
			}

			ticksUsed, done := procToRun.runTillOutOrDone(ticksLeftPerCore[currCore])

			toWrite := fmt.Sprintf("%v, %v, %v, running proc %v, gave %v ticks, used %v ticks\n", int(*sd.currTickPtr), sd.machineId, currCore, procToRun.String(), ticksLeftPerCore[currCore].String(), ticksUsed.String())
			logWrite(SCHED, toWrite)

			ticksLeftPerCore[currCore] -= ticksUsed
			ticksLeftToGive -= ticksUsed
			totalTicksLeftToGive -= ticksUsed

			if !done {
				sd.activeQ[Tid(currCore)].enq(procToRun)
			} else {
				// if the proc is done, update the ticksPassed to be exact for metrics etc
				procToRun.timeDone = *sd.currTickPtr + (1 - ticksLeftPerCore[currCore])

				if (procToRun.timeDone - procToRun.timeStarted) > procToRun.deadline {
					toWrite := fmt.Sprintf("PROC OVER: %v \n", procToRun.String())
					logWrite(SCHED, toWrite)
				}

				toWrite := fmt.Sprintf("%v, %v, %v, %v, %v, %v\n", int(*sd.currTickPtr), procToRun.machineId, procToRun.procInternals.procType, float64(procToRun.deadline), float64(procToRun.timeDone-procToRun.timeStarted), float64(procToRun.procInternals.actualComp))
				logWrite(DONE_PROCS, toWrite)
			}
		}
	}

	// do this for every core
	for coreNum := 0; coreNum < sd.numCores; coreNum++ {
		// use core num to get info
		if sd.activeQ[Tid(coreNum)].getHOLSlack(*sd.currTickPtr) > IDLE_HEAP_THRESHOLD {

			toWrite := fmt.Sprintf("adding machine %d core %v to idle \n", sd.machineId, coreNum)
			logWrite(SCHED, toWrite)

			sd.idleHeap.lock.Lock()
			// also if it is already in the heap, then replace it with the new value
			if contains(sd.idleHeap.heap, TmachineCoreId{sd.machineId, Tid(coreNum)}) {
				remove(sd.idleHeap.heap, TmachineCoreId{sd.machineId, Tid(coreNum)})
			}
			toPush := TIdleMachine{
				compIdleFor:   sd.activeQ[Tid(coreNum)].getHOLSlack(*sd.currTickPtr),
				machineCoreId: TmachineCoreId{sd.machineId, Tid(coreNum)},
			}
			sd.idleHeap.heap.Push(toPush)
			sd.idleHeap.lock.Unlock()
		}
	}

	// this is dumb but make accounting for util easier
	if totalTicksLeftToGive < 0.00002 {
		totalTicksLeftToGive = 0
	}
	toWrite = fmt.Sprintf(", %v\n", float64(math.Copysign(float64(totalTicksLeftToGive), 1)))
	logWrite(USAGE, toWrite)
}
