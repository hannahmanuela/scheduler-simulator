package slasched

import (
	"fmt"
	"math"
)

const (
	MONEY_WASTE_THRESHOLD = 0.5
)

type IdealDC struct {
	currTickPtr             *Tftick
	procQ                   *Queue
	amtWorkPerTick          int
	totalMem                Tmem
	worldNumProcsGenPerTick int
}

func newIdealDC(amtWorkPerTick int, totMem Tmem, currTickPtr *Tftick, worldNumProcsGenPerTick int) *IdealDC {
	return &IdealDC{
		currTickPtr:             currTickPtr,
		procQ:                   newQueue(),
		amtWorkPerTick:          amtWorkPerTick,
		totalMem:                totMem,
		worldNumProcsGenPerTick: worldNumProcsGenPerTick,
	}

}

func (idc *IdealDC) memFree() Tmem {
	currMemUsed := Tmem(0)

	for _, p := range idc.procQ.getQ() {
		currMemUsed += p.maxMem()
	}

	return idc.totalMem - currMemUsed
}

func (idc *IdealDC) potPlaceProc(newProc *Proc) bool {

	// if it just fits in terms of memory do it
	if newProc.maxMem() < idc.memFree() {

		newProc.timePlaced = *idc.currTickPtr
		idc.procQ.enq(newProc)
		return true
	}

	// if it doesn't fit, look if there a good proc to kill? (/a combination of procs? can add that later)
	procToKill, minMoneyWaste := idc.procQ.checkKill(newProc)
	if minMoneyWaste < MONEY_WASTE_THRESHOLD {

		newProc.timePlaced = *idc.currTickPtr
		idc.procQ.kill(procToKill)
		idc.procQ.enq(newProc)
		return true
	}

	return false

}

// ok so I have a bunch of procs that all fit memory wise, so really what I'm doing
func (idc *IdealDC) tick() {

	toWrite := fmt.Sprintf("%v @ %v; mem free: %v: WHOLE QUEUE %v\n", idc.worldNumProcsGenPerTick, idc.currTickPtr, idc.memFree(), idc.procQ.String())
	logWrite(IDEAL_SCHED, toWrite)

	totalTicksLeftToGive := Tftick(idc.amtWorkPerTick)
	ticksLeftPerCore := make(map[int]Tftick, 0)
	coresWithTicksLeft := make(map[int]bool, 0)
	coresLeftThisRound := make(map[int]bool, 0)

	for i := 0; i < idc.amtWorkPerTick; i++ {
		ticksLeftPerCore[i] = Tftick(1)
		coresWithTicksLeft[i] = true
	}

	ogMemFree := idc.memFree()
	toWrite = fmt.Sprintf("%v, %v", idc.worldNumProcsGenPerTick, int(*idc.currTickPtr))
	logWrite(IDEAL_USAGE, toWrite)

	// TODO: what if it doesn't fit?
	putProcOnCoreWithMaxTimeLeft := func() int {
		maxVal := Tftick(0.0)
		coreToUse := -1
		for i := 0; i < idc.amtWorkPerTick; i++ {
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

	for idc.procQ.qlen() > 0 && totalTicksLeftToGive-Tftick(TICK_SCHED_THRESHOLD) > 0.0 && len(coresWithTicksLeft) > 0 {

		for i := 0; i < idc.amtWorkPerTick; i++ {
			coresLeftThisRound[i] = true
		}

		// run by amount of money willing to spend
		coreToProc := make(map[int]*Proc, idc.amtWorkPerTick)
		for i := 0; i < idc.amtWorkPerTick; i++ {
			p := idc.procQ.deq()
			if p == nil {
				continue
			}
			coreToUse := putProcOnCoreWithMaxTimeLeft()
			if coreToUse == -1 {
				idc.procQ.enq(p)
				coreToProc[coreToUse] = nil
			} else {
				coreToProc[coreToUse] = p
			}
		}

		// run all the cores once
		for currCore := 0; currCore < idc.amtWorkPerTick; currCore++ {

			procToRun := coreToProc[currCore]

			if procToRun == nil {
				continue
			}

			toWrite := fmt.Sprintf("   core %v giving %v to proc %v \n", currCore, ticksLeftPerCore[currCore], procToRun.String())
			logWrite(IDEAL_SCHED, toWrite)

			ticksUsed, done := procToRun.runTillOutOrDone(ticksLeftPerCore[currCore])

			ticksLeftPerCore[currCore] -= ticksUsed
			totalTicksLeftToGive -= ticksUsed

			if ticksLeftPerCore[currCore] < Tftick(TICK_SCHED_THRESHOLD) {
				delete(coresWithTicksLeft, currCore)
			}

			if !done {
				toReq = append(toReq, procToRun)
			} else {
				// if the proc is done, update the ticksPassed to be exact for metrics etc
				procToRun.timeDone = *idc.currTickPtr + (1 - ticksLeftPerCore[currCore])

				toWrite := fmt.Sprintf("   -> done: %v\n", procToRun.String())
				logWrite(IDEAL_SCHED, toWrite)

				if (procToRun.timeDone - procToRun.timeStarted) > procToRun.compDone {
					toWrite := fmt.Sprintf("   ---> OVER %v \n", procToRun.String())
					logWrite(IDEAL_SCHED, toWrite)
				}

				toWrite = fmt.Sprintf("%v, %v, %v, %v \n", idc.worldNumProcsGenPerTick, procToRun.willingToSpend(), (procToRun.timeDone - procToRun.timeStarted).String(), procToRun.compDone.String())
				logWrite(IDEAL_PROCS_DONE, toWrite)
			}

		}

	}

	for _, p := range toReq {
		idc.procQ.enq(p)
	}

	toWrite = fmt.Sprintf("cores with ticks left: %v, ticks left over: %v\n", coresWithTicksLeft, ticksLeftPerCore)
	logWrite(IDEAL_SCHED, toWrite)

	if totalTicksLeftToGive < 0.00002 {
		totalTicksLeftToGive = 0
	}
	toWrite = fmt.Sprintf(", %v, %v\n", float64(math.Copysign(float64(totalTicksLeftToGive), 1)), ogMemFree)
	logWrite(IDEAL_USAGE, toWrite)
}
