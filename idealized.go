package slasched

import (
	"fmt"
	"math"
	"sort"
)

type IdealDC struct {
	currTickPtr             *Tftick
	procQ                   *Queue
	amtWorkPerTick          int
	worldNumProcsGenPerTick int
}

func newIdealDC(amtWorkPerTick int, currTickPtr *Tftick, worldNumProcsGenPerTick int) *IdealDC {
	return &IdealDC{
		currTickPtr:             currTickPtr,
		procQ:                   newQueue(),
		amtWorkPerTick:          amtWorkPerTick,
		worldNumProcsGenPerTick: worldNumProcsGenPerTick,
	}

}

func (idc *IdealDC) addProc(newProc *Proc) {
	idc.procQ.enq(newProc)
}

func (idc *IdealDC) okToPlace(newProc *Proc) bool {

	fullList := append(make([]*Proc, 0, len(idc.procQ.getQ())+1), idc.procQ.getQ()...)
	fullList = append(fullList, newProc)
	sort.Slice(fullList, func(i, j int) bool {
		return fullList[i].deadline < fullList[j].deadline
	})

	coresToRunningWaitTime := make(map[int]Tftick)
	for i := 0; i < idc.amtWorkPerTick; i++ {
		coresToRunningWaitTime[i] = 0
	}

	getAddMinRunningWaitTime := func(toAdd Tftick) Tftick {
		minVal := Tftick(math.MaxFloat32)
		minCore := -1
		for i := 0; i < idc.amtWorkPerTick; i++ {
			if coresToRunningWaitTime[i] < minVal {
				minVal = coresToRunningWaitTime[i]
				minCore = i
			}
		}
		coresToRunningWaitTime[minCore] += toAdd
		return minVal
	}

	for _, p := range fullList {

		waitTime := getAddMinRunningWaitTime(p.getMaxCompLeft())
		if float64(p.getSlack(*idc.currTickPtr)-waitTime) < 0.0 {
			return false
		}
	}

	return true
}

func (idc *IdealDC) tick() {

	totalTicksLeftToGive := Tftick(idc.amtWorkPerTick)
	ticksLeftPerCore := make(map[int]Tftick, 0)
	coresLeft := make(map[int]bool, 0)

	for i := 0; i < idc.amtWorkPerTick; i++ {
		ticksLeftPerCore[i] = Tftick(1)
		coresLeft[i] = true
	}

	toWrite := fmt.Sprintf("%v, %v, %v, %v", idc.worldNumProcsGenPerTick, int(*idc.currTickPtr), -1, idc.procQ.qlen())
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

		// distribute rest of procs among cores
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

			ticksUsed, done := procToRun.runTillOutOrDone(ticksLeftPerCore[currCore])

			ticksLeftPerCore[currCore] -= ticksUsed
			totalTicksLeftToGive -= ticksUsed

			if !done {
				// TODO: check this
				toReq = append(toReq, procToRun)
			} else {
				// if the proc is done, update the ticksPassed to be exact for metrics etc
				procToRun.timeDone = *idc.currTickPtr + (1 - ticksLeftPerCore[currCore])

				if procToRun.timeDone-procToRun.timeStarted > procToRun.deadline {
					fmt.Printf("IDEAL PROC OVER: %v, time done: %v\n", procToRun.String(), procToRun.timeDone)
				}
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

// for currCore := 0; currCore < idc.amtWorkPerTick; currCore++ {

// 	ticksLeftToGive := Tftick(1)
// 	ranDesignated := false

// inner:
// 	for ticksLeftToGive-Tftick(TICK_SCHED_THRESHOLD) > 0.0 {

// 		var procToRun *Proc
// 		if !ranDesignated {
// 			procToRun = coreToProc[currCore]
// 			ranDesignated = true
// 		} else {
// 			procToRun = idc.procQ.deq()
// 		}

// 		if procToRun == nil {
// 			break inner
// 		}

// 		ticksUsed, done := procToRun.runTillOutOrDone(ticksLeftToGive)

// 		ticksLeftToGive -= ticksUsed
// 		totalTicksLeftToGive -= ticksUsed

// 		if !done {
// 			// this works because the proc ran up until the end of the tick, so nothing else should be allowed to work steal it
// 			toReq = append(toReq, procToRun)
// 		} else {
// 			// if the proc is done, update the ticksPassed to be exact for metrics etc
// 			procToRun.timeDone = *idc.currTickPtr + (1 - ticksLeftToGive)

// 			if procToRun.timeDone-procToRun.timeStarted > procToRun.deadline {
// 				fmt.Printf("IDEAL PROC OVER: %v, time done: %v\n", procToRun.String(), procToRun.timeDone)
// 			}
// 		}
// 	}
// }
