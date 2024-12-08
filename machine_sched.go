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
	activeQ                 map[Tid]*Queue
	currTickPtr             *Tftick
	worldNumProcsGenPerTick int
}

func newSched(numCores int, mid Tid, currTickPtr *Tftick, nGenPerTick int) *Sched {
	sd := &Sched{
		machineId:               mid,
		numCores:                numCores,
		activeQ:                 make(map[Tid]*Queue),
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
		for _, p := range sd.activeQ[Tid(i)].getAllProcs() {
			toWrite := fmt.Sprintf("%v, %v, 1, %v, %v, %v\n", int(*sd.currTickPtr), sd.machineId,
				float64(p.priority), float64(p.procInternals.actualComp), float64(p.compDone))
			logWrite(CURR_PROCS, toWrite)
		}
	}
}

func (sd *Sched) placeProc(newProc *Proc, coreId Tid) {

	sd.activeQ[coreId].enq(newProc)

}

func (sd *Sched) tryPlace(newProc *Proc) (Tftick, Tid) {

	// TODO:

	// check if the tenant has the tokens for it currently?

	return 0, 0

}

func (sd *Sched) simulateRunProcs() {

	sum_qlens := 0
	for i := 0; i < sd.numCores; i++ {
		sum_qlens += sd.activeQ[Tid(i)].numProcs()
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

		for ticksLeftToGive-Tftick(TICK_SCHED_THRESHOLD) > 0.0 && sd.activeQ[Tid(currCore)].numProcs() > 0 {

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

				toWrite := fmt.Sprintf("%v, %v, %v, %v, %v\n", int(*sd.currTickPtr), procToRun.procInternals.procType, float64(procToRun.priority), float64(procToRun.timeDone-procToRun.timeStarted), float64(procToRun.procInternals.actualComp))
				logWrite(DONE_PROCS, toWrite)
			}
		}
	}

	// this is dumb but make accounting for util easier
	if totalTicksLeftToGive < 0.00002 {
		totalTicksLeftToGive = 0
	}
	toWrite = fmt.Sprintf(", %v\n", float64(math.Copysign(float64(totalTicksLeftToGive), 1)))
	logWrite(USAGE, toWrite)
}
