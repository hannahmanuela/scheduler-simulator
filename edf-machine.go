package slasched

import (
	"fmt"
	"math"
)

type BigEDFMachine struct {
	currTickPtr             *Tftick
	procs                   []*EDFProc
	amtWorkPerTick          int
	totalMem                Tmem
	worldNumProcsGenPerTick int
}

func newBigEDFMachine(amtWorkPerTick int, totMem Tmem, currTickPtr *Tftick, worldNumProcsGenPerTick int) *BigEDFMachine {
	return &BigEDFMachine{
		currTickPtr:             currTickPtr,
		procs:                   make([]*EDFProc, 0),
		amtWorkPerTick:          amtWorkPerTick,
		totalMem:                totMem,
		worldNumProcsGenPerTick: worldNumProcsGenPerTick,
	}

}

func (edfm *BigEDFMachine) placeProc(newProc *EDFProc) {

	edfm.enq(newProc)

}

func (edfm *BigEDFMachine) memFree() Tmem {

	memUsed := Tmem(0)

	for _, p := range edfm.procs {
		memUsed += p.p.memUsing
	}
	return edfm.totalMem - memUsed
}

func (edfm *BigEDFMachine) tick() {

	toWrite := fmt.Sprintf("%v @ %v; mem free: %v: WHOLE QUEUE ", edfm.worldNumProcsGenPerTick, edfm.currTickPtr, MEM_PER_MACHINE)
	logWrite(EDF_SCHED, toWrite)
	for _, p := range edfm.procs {
		toWrite := fmt.Sprintf("%v, dl: %.2f; \n", p.String(), p.dl)
		logWrite(EDF_SCHED, toWrite)
	}
	logWrite(EDF_SCHED, "\n")

	totalTicksLeftToGive := Tftick(edfm.amtWorkPerTick)
	ticksLeftPerCore := make(map[int]Tftick, 0)
	coresWithTicksLeft := make(map[int]bool, 0)
	coresLeftThisRound := make(map[int]bool, 0)

	for i := 0; i < edfm.amtWorkPerTick; i++ {
		ticksLeftPerCore[i] = Tftick(1)
		coresWithTicksLeft[i] = true
	}

	ogMemFree := edfm.memFree()
	toWrite = fmt.Sprintf("%v, %v", edfm.worldNumProcsGenPerTick, int(*edfm.currTickPtr))
	logWrite(EDF_USAGE, toWrite)

	// TODO: what if it doesn't fit?
	putProcOnCoreWithMaxTimeLeft := func() int {
		maxVal := Tftick(0.0)
		coreToUse := -1
		for i := 0; i < edfm.amtWorkPerTick; i++ {
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

	toReq := make([]*EDFProc, 0)

	for len(edfm.procs) > 0 && totalTicksLeftToGive-Tftick(TICK_SCHED_THRESHOLD) > 0.0 && len(coresWithTicksLeft) > 0 {

		for i := 0; i < edfm.amtWorkPerTick; i++ {
			coresLeftThisRound[i] = true
		}

		coreToProc := make(map[int]*EDFProc, edfm.amtWorkPerTick)
		for i := 0; i < edfm.amtWorkPerTick; i++ {
			p := edfm.deq()
			if p == nil {
				continue
			}
			coreToUse := putProcOnCoreWithMaxTimeLeft()
			if coreToUse == -1 {
				edfm.enq(p)
				coreToProc[coreToUse] = nil
			} else {
				coreToProc[coreToUse] = p
			}
		}

		// run all the cores once
		for currCore := 0; currCore < edfm.amtWorkPerTick; currCore++ {

			procToRun := coreToProc[currCore]

			if procToRun == nil {
				continue
			}

			toWrite := fmt.Sprintf("   core %v giving %v to proc %v \n", currCore, ticksLeftPerCore[currCore], procToRun.String())
			logWrite(EDF_SCHED, toWrite)

			_, ticksUsed, done := procToRun.p.runTillOutOrDone(ticksLeftPerCore[currCore])

			ticksLeftPerCore[currCore] -= ticksUsed
			totalTicksLeftToGive -= ticksUsed

			if ticksLeftPerCore[currCore] < Tftick(TICK_SCHED_THRESHOLD) {
				delete(coresWithTicksLeft, currCore)
			}

			if !done {
				toReq = append(toReq, procToRun)
			} else {
				// if the proc is done, update the ticksPassed to be exact for metrics etc
				procToRun.p.timeDone = *edfm.currTickPtr + (1 - ticksLeftPerCore[currCore])

				toWrite := fmt.Sprintf("   -> done: %v\n", procToRun.String())
				logWrite(EDF_SCHED, toWrite)

				if (procToRun.p.timeDone - procToRun.p.timeStarted) > procToRun.p.compDone {
					toWrite := fmt.Sprintf("   ---> OVER %v \n", procToRun.String())
					logWrite(EDF_SCHED, toWrite)
				}

				toWrite = fmt.Sprintf("%v, %v, %v, %v \n", edfm.worldNumProcsGenPerTick, procToRun.p.willingToSpend(), (procToRun.p.timeDone - procToRun.p.timeStarted).String(), procToRun.p.compDone.String())
				logWrite(EDF_PROCS_DONE, toWrite)
			}

		}

	}

	for _, p := range toReq {
		edfm.enq(p)
	}

	toWrite = fmt.Sprintf("cores with ticks left: %v, ticks left over: %v\n", coresWithTicksLeft, ticksLeftPerCore)
	logWrite(EDF_SCHED, toWrite)

	if totalTicksLeftToGive < 0.00002 {
		totalTicksLeftToGive = 0
	}
	toWrite = fmt.Sprintf(", %.3f, %v\n", float64(math.Copysign(float64(totalTicksLeftToGive), 1)), ogMemFree)
	logWrite(EDF_USAGE, toWrite)
}

func (edfm *BigEDFMachine) deq() *EDFProc {
	minDl := float32(math.MaxFloat32)
	var procToRet *EDFProc
	idxToDel := -1

	for i, p := range edfm.procs {
		if p.dl < minDl {
			minDl = p.dl
			procToRet = p
			idxToDel = i
		}
	}

	if idxToDel >= 0 {
		edfm.procs = append(edfm.procs[:idxToDel], edfm.procs[idxToDel+1:]...)
	}

	return procToRet
}

func (edfm *BigEDFMachine) enq(newProc *EDFProc) {

	edfm.procs = append(edfm.procs, newProc)
}
