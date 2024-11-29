package slasched

import (
	"sort"
)

type IdealDC struct {
	currTickPtr    *Tftick
	procQ          *Queue
	amtWorkPerTick int
}

func newIdealDC(amtWorkPerTick int, currTickPtr *Tftick) *IdealDC {

	return &IdealDC{
		currTickPtr:    currTickPtr,
		procQ:          newQueue(),
		amtWorkPerTick: amtWorkPerTick,
	}

}

func (idc *IdealDC) addProc(newProc *Proc) {
	idc.procQ.enq(newProc)
}

func (idc *IdealDC) okToPlace(newProc *Proc) bool {

outer:
	for currCore := 0; currCore < idc.amtWorkPerTick; currCore++ {

		fullList := append(make([]*Proc, 0, len(idc.procQ.getQ())+1), idc.procQ.getQ()...)
		fullList = append(fullList, newProc)
		sort.Slice(fullList, func(i, j int) bool {
			return fullList[i].deadline < fullList[j].deadline
		})

		runningWaitTime := Tftick(0)

		for _, p := range fullList {

			if float64(p.getSlack(*idc.currTickPtr)-runningWaitTime) < 0.0 {
				continue outer
			}
			runningWaitTime += p.getExpectedCompLeft()
		}

		return true

	}

	return false
}

func (idc *IdealDC) tick() {

	coreToProc := make(map[int]*Proc, 0)

	toReq := make([]*Proc, 0)

	for i := 0; i < idc.amtWorkPerTick; i++ {
		coreToProc[i] = idc.procQ.deq()
	}

	ticksLeftPerCore := make(map[int]Tftick, 0)
	totalTicksLeftToGive := Tftick(idc.amtWorkPerTick)

	for currCore := 0; currCore < idc.amtWorkPerTick; currCore++ {

		ticksLeftToGive := Tftick(1)
		ticksLeftPerCore[currCore] = Tftick(1)
		ranDesignated := false

		for ticksLeftToGive-Tftick(TICK_SCHED_THRESHOLD) > 0.0 && idc.procQ.qlen() > 0 {

			var procToRun *Proc
			if !ranDesignated {
				procToRun = coreToProc[currCore]
				ranDesignated = true
			} else {
				procToRun = idc.procQ.deq()
			}

			if procToRun == nil {
				break
			}

			ticksUsed, done := procToRun.runTillOutOrDone(ticksLeftPerCore[currCore])

			ticksLeftPerCore[currCore] -= ticksUsed
			ticksLeftToGive -= ticksUsed
			totalTicksLeftToGive -= ticksUsed

			if !done {
				// this works because the proc ran up until the end of the tick, so nothing else should be allowed to work steal it
				toReq = append(toReq, procToRun)
			} else {
				// if the proc is done, update the ticksPassed to be exact for metrics etc
				procToRun.timeDone = *idc.currTickPtr + (1 - ticksLeftPerCore[currCore])
			}
		}
	}

	for _, p := range toReq {
		idc.procQ.enq(p)
	}
}
