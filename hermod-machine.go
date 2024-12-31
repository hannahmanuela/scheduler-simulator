package slasched

import (
	"fmt"
	"math"
)

// what it's going to need to do:

// take in procs [NOT stored in a queue]

// run numCores ticks over them -- using PS

// report util, latency

type HermodMachine struct {
	machineId               Tid
	currTickPtr             *Tftick
	numCores                int
	procQ                   []*Proc
	totalMem                Tmem
	worldNumProcsGenPerTick int
}

func newHermodMachine(mid Tid, numCores int, totMem Tmem, currTickPtr *Tftick, worldNumProcsGenPerTick int) *HermodMachine {
	return &HermodMachine{
		machineId:               mid,
		currTickPtr:             currTickPtr,
		numCores:                numCores,
		procQ:                   make([]*Proc, 0),
		totalMem:                totMem,
		worldNumProcsGenPerTick: worldNumProcsGenPerTick,
	}

}

func (hm *HermodMachine) memFree() Tmem {
	currMemUsed := Tmem(0)

	for _, p := range hm.procQ {
		currMemUsed += p.maxMem()
	}

	return hm.totalMem - currMemUsed
}

func (hm *HermodMachine) placeProc(newProc *Proc) {

	newProc.timePlaced = *hm.currTickPtr
	hm.procQ = append(hm.procQ, newProc)

}

func (hm *HermodMachine) tick() {

	// do PS across all the procs
	ticksLeftPerCore := make(map[int]Tftick, hm.numCores)
	totalTicksLeftToGive := Tftick(hm.numCores)

	for i := 0; i < hm.numCores; i++ {
		ticksLeftPerCore[i] = Tftick(1)
	}

	// water-filling: assign procs to cores
	procsPerCore := make(map[int][]*Proc)
	currCore := 0
	for _, p := range hm.procQ {
		procsPerCore[currCore] = append(procsPerCore[currCore], p)
		currCore += 1
		if currCore == hm.numCores {
			currCore = 0
		}
	}

	ogMemFree := hm.memFree()
	toWrite := fmt.Sprintf("%v, %v, %v", hm.worldNumProcsGenPerTick, int(*hm.currTickPtr), hm.machineId)
	logWrite(HERMOD_USAGE, toWrite)

	toWrite = fmt.Sprintf("\n==> %v @ %v, machine %v, mem free: %v, has q: \n%v", hm.worldNumProcsGenPerTick, hm.currTickPtr.String(), hm.machineId, hm.memFree(), hm.procQ)
	logWrite(HERMOD_SCHED, toWrite)

	// water-filling, assign ticks to procs
	for currCore := 0; currCore < hm.numCores; currCore++ {

		for len(procsPerCore[currCore]) > 0 && ticksLeftPerCore[currCore]-Tftick(TICK_SCHED_THRESHOLD) > 0.0 {

			ticksToGive := ticksLeftPerCore[currCore] / Tftick(len(procsPerCore[currCore]))
			currProc := procsPerCore[currCore][0]
			procsPerCore[currCore] = procsPerCore[currCore][1:]

			ticksUsed, done := currProc.runTillOutOrDone(ticksToGive)

			ticksLeftPerCore[currCore] -= ticksUsed
			totalTicksLeftToGive -= ticksUsed

			if done {

				currProc.timeDone = *hm.currTickPtr + (1 - ticksLeftPerCore[currCore])

				toWrite = fmt.Sprintf("%v, %v, %v, %v \n", hm.worldNumProcsGenPerTick, currProc.willingToSpend(), (currProc.timeDone - currProc.timeStarted).String(), currProc.compDone.String())
				logWrite(HERMOD_PROCS_DONE, toWrite)

				hm.removeProcFromQ(currProc)
			}
		}
	}

	toWrite = fmt.Sprintf(", %v, %v\n", float64(math.Copysign(float64(totalTicksLeftToGive), 1)), ogMemFree)
	logWrite(HERMOD_USAGE, toWrite)

}

func (hm *HermodMachine) removeProcFromQ(procToRemove *Proc) {

	newQ := make([]*Proc, len(hm.procQ)-1)

	for i, p := range hm.procQ {
		if p == procToRemove {
			newQ = append(hm.procQ[:i], hm.procQ[i+1:]...)
		}
	}

	hm.procQ = newQ

}
