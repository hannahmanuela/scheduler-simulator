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
		idc.procQ.enq(newProc)
		return true
	}

	// if it doesn't fit, look if there a good proc to kill? (/a combination of procs? can add that later)
	procToKill, minMoneyWaste := idc.procQ.checkKill(newProc)
	if procToKill > 0 {
		if minMoneyWaste < MONEY_WASTE_THRESHOLD {
			idc.procQ.kill(procToKill)
			return true
		}
	}

	return false

}

// ok so I have a bunch of procs that all fit memory wise, so really what I'm doing
func (idc *IdealDC) tick() {

	toWrite := fmt.Sprintf("%v @ %v: WHOLE QUEUE %v\n", idc.worldNumProcsGenPerTick, idc.currTickPtr, idc.procQ.String())
	logWrite(IDEAL_SCHED, toWrite)

	totalTicksLeftToGive := Tftick(idc.amtWorkPerTick)
	ticksLeftPerCore := make(map[int]Tftick, 0)
	coresLeft := make(map[int]bool, 0)

	for i := 0; i < idc.amtWorkPerTick; i++ {
		ticksLeftPerCore[i] = Tftick(1)
		coresLeft[i] = true
	}

	toWrite = fmt.Sprintf("%v, %v", idc.worldNumProcsGenPerTick, int(*idc.currTickPtr))
	logWrite(IDEAL_USAGE, toWrite)

	putProcOnCoreWithMaxTimeLeft := func() int {
		minVal := Tftick(math.MaxFloat32)
		minCore := -1
		for i := 0; i < idc.amtWorkPerTick; i++ {
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

	for idc.procQ.qlen() > 0 && totalTicksLeftToGive-Tftick(TICK_SCHED_THRESHOLD) > 0.0 {

		// run by amount of money willing to spend
		coreToProc := make(map[int]*Proc, 0)
		for i := 0; i < idc.amtWorkPerTick; i++ {
			p := idc.procQ.deq()
			if p == nil {
				continue
			}
			coreToUse := putProcOnCoreWithMaxTimeLeft()
			coreToProc[coreToUse] = p
		}

		// run all the cores once
		for currCore := 0; currCore < idc.amtWorkPerTick; currCore++ {

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
				procToRun.timeDone = *idc.currTickPtr + (1 - ticksLeftPerCore[currCore])

				toWrite := fmt.Sprintf("%v, %.2f, %.2f, %.2f \n", idc.worldNumProcsGenPerTick, procToRun.willingToSpend(), float32(procToRun.timeDone-procToRun.timeStarted), float32(procToRun.compDone))
				logWrite(IDEAL_PROCS_DONE, toWrite)
			}

		}

	}

	for _, p := range toReq {
		idc.procQ.enq(p)
	}

	if totalTicksLeftToGive < 0.00002 {
		totalTicksLeftToGive = 0
	}
	toWrite = fmt.Sprintf(", %v\n", float64(math.Copysign(float64(totalTicksLeftToGive), 1)))
	logWrite(IDEAL_USAGE, toWrite)
}
