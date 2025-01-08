package slasched

import (
	"fmt"
	"math"
)

type HermodMachine struct {
	machineId               Tid
	currTickPtr             *Tftick
	numCores                int
	procs                   []*Proc
	totalMem                Tmem
	worldNumProcsGenPerTick int
}

func newHermodMachine(mid Tid, numCores int, totMem Tmem, currTickPtr *Tftick, worldNumProcsGenPerTick int) *HermodMachine {
	return &HermodMachine{
		machineId:               mid,
		currTickPtr:             currTickPtr,
		numCores:                numCores,
		procs:                   make([]*Proc, 0),
		totalMem:                totMem,
		worldNumProcsGenPerTick: worldNumProcsGenPerTick,
	}

}

func (hm *HermodMachine) memFree() Tmem {
	currMemUsed := Tmem(0)

	for _, p := range hm.procs {
		currMemUsed += p.memUsing
	}

	return hm.totalMem - currMemUsed
}

func (hm *HermodMachine) placeProc(newProc *Proc) {

	newProc.timePlaced = *hm.currTickPtr
	hm.procs = append(hm.procs, newProc)

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
	for _, p := range hm.procs {
		procsPerCore[currCore] = append(procsPerCore[currCore], p)
		currCore += 1
		if currCore == hm.numCores {
			currCore = 0
		}
	}

	ogMemFree := hm.memFree()
	toWrite := fmt.Sprintf("%v, %v, %v", hm.worldNumProcsGenPerTick, int(*hm.currTickPtr), hm.machineId)
	logWrite(HERMOD_USAGE, toWrite)

	toWrite = fmt.Sprintf("\n==> %v @ %v, machine %v, mem free: %v, has q: \n%v", hm.worldNumProcsGenPerTick, hm.currTickPtr.String(), hm.machineId, hm.memFree(), hm.procs)
	logWrite(HERMOD_SCHED, toWrite)

	// water-filling, assign ticks to procs
	for currCore := 0; currCore < hm.numCores; currCore++ {

		for len(procsPerCore[currCore]) > 0 && ticksLeftPerCore[currCore]-Tftick(TICK_SCHED_THRESHOLD) > 0.0 {

			ticksToGive := ticksLeftPerCore[currCore] / Tftick(len(procsPerCore[currCore]))
			currProc := procsPerCore[currCore][0]
			procsPerCore[currCore] = procsPerCore[currCore][1:]

			_, ticksUsed, done := currProc.runTillOutOrDone(ticksToGive)

			ticksLeftPerCore[currCore] -= ticksUsed
			totalTicksLeftToGive -= ticksUsed

			if done {

				currProc.timeDone = *hm.currTickPtr + (1 - ticksLeftPerCore[currCore])

				toWrite = fmt.Sprintf("%v, %v, %v, %v \n", hm.worldNumProcsGenPerTick, currProc.willingToSpend(), (currProc.timeDone - currProc.timeStarted).String(), currProc.compDone.String())
				logWrite(HERMOD_PROCS_DONE, toWrite)

				hm.removeProc(currProc)
			}
		}
	}

	if totalTicksLeftToGive < 0.00002 {
		totalTicksLeftToGive = 0
	}

	toWrite = fmt.Sprintf(", %.3f, %v\n", float64(math.Copysign(float64(totalTicksLeftToGive), 1)), ogMemFree)
	logWrite(HERMOD_USAGE, toWrite)

}

func (hm *HermodMachine) removeProc(procToRemove *Proc) {

	newQ := make([]*Proc, len(hm.procs)-1)

	for i, p := range hm.procs {
		if p == procToRemove {
			newQ = append(hm.procs[:i], hm.procs[i+1:]...)
		}
	}

	hm.procs = newQ

}
