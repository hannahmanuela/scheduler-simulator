package slasched

import (
	"fmt"
	"math"
)

type IdealDC struct {
	currTickPtr             *Tftick
	procQs                  map[*Ttenant]*Queue
	amtWorkPerTick          int
	worldNumProcsGenPerTick int
}

func newIdealDC(amtWorkPerTick int, currTickPtr *Tftick, worldNumProcsGenPerTick int, tenants []*Ttenant) *IdealDC {
	idc := &IdealDC{
		currTickPtr:             currTickPtr,
		amtWorkPerTick:          amtWorkPerTick,
		worldNumProcsGenPerTick: worldNumProcsGenPerTick,
	}
	idc.procQs = make(map[*Ttenant]*Queue, len(tenants))
	for _, tn := range tenants {
		idc.procQs[tn] = newQueue()
	}
	return idc
}

func (idc *IdealDC) addProc(newProc *Proc) {

	for tn, q := range idc.procQs {
		if tn.id == newProc.tenantId {
			q.enq(newProc)
			return
		}
	}

}

func (idc *IdealDC) tick() {

	toWrite := fmt.Sprintf("%v @ %v: \n", idc.worldNumProcsGenPerTick, idc.currTickPtr)
	logWrite(IDEAL_SCHED, toWrite)
	for id, q := range idc.procQs {
		toWrite := fmt.Sprintf("  %v - %v \n", id, q.String())
		logWrite(IDEAL_SCHED, toWrite)
	}

	toReq := make(map[*Ttenant][]*Proc, 0)

	for tn, q := range idc.procQs {

		for tn.currNumCompTokens-Tftick(TICK_SCHED_THRESHOLD) > 0.0 {

			ticksToGive := Tftick(math.Min(float64(tn.currNumCompTokens), 1))
			procToRun := q.deq()
			compUsed, done := procToRun.runTillOutOrDone(ticksToGive)
			tn.currNumCompTokens -= compUsed

			if !done {
				if _, ok := toReq[tn]; ok {
					toReq[tn] = append(toReq[tn], procToRun)
				} else {
					toReq[tn] = []*Proc{procToRun}
				}
			}

		}

	}

	// if there are tokens left over somewhere, that means a tenant didn't have enough load to use all of their tokens
	// allow others to use the tokens they had left over
	toksLeftOver := Tftick(0)
	for tn, _ := range idc.procQs {
		toksLeftOver += tn.currNumCompTokens
	}

	// TODO: note which tenants are giving/taking
	// have a balance that gets added to/taken from?

	for toksLeftOver-Tftick(TICK_SCHED_THRESHOLD) > 0.0 {
		// this as it stands is just round robin
		foundOne := false
		for tn, q := range idc.procQs {
			if q.numProcs() > 0 {
				foundOne = true

				ticksToGive := Tftick(math.Min(float64(toksLeftOver), 1))
				procToRun := q.deq()
				compUsed, done := procToRun.runTillOutOrDone(ticksToGive)
				toksLeftOver -= compUsed

				if !done {
					if _, ok := toReq[tn]; ok {
						toReq[tn] = append(toReq[tn], procToRun)
					} else {
						toReq[tn] = []*Proc{procToRun}
					}
				}
			}
		}
		if !foundOne {
			break
		}
	}

	for tn, lp := range toReq {
		for _, p := range lp {
			idc.procQs[tn].enq(p)
		}
	}

	toWrite = fmt.Sprintf("%v, %v", idc.worldNumProcsGenPerTick, int(*idc.currTickPtr))
	logWrite(IDEAL_USAGE, toWrite)

	if toksLeftOver < 0.00002 {
		toksLeftOver = 0
	}
	toWrite = fmt.Sprintf(", %v\n", float64(math.Copysign(float64(toksLeftOver), 1)))
	logWrite(IDEAL_USAGE, toWrite)
}
