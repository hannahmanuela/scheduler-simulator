package slasched

type IdealDC struct {
	procQ          *Queue
	amtWorkPerTick int
}

func newIdealDC(amtWorkPerTick int) *IdealDC {

	return &IdealDC{
		procQ:          newQueue(),
		amtWorkPerTick: amtWorkPerTick,
	}

}

func (idc *IdealDC) addProc(newProc *Proc) {
	idc.procQ.enq(newProc)
}

func (idc *IdealDC) tick() {

	ticksLeftToGive := Tftick(idc.amtWorkPerTick)

	for ticksLeftToGive-Tftick(TICK_SCHED_THRESHOLD) > 0.0 && idc.procQ.qlen() > 0 {

		procToRun := idc.procQ.deq()

		if procToRun == nil {
			break
		}

		ticksUsed, done := procToRun.runTillOutOrDone(ticksLeftToGive)

		ticksLeftToGive -= ticksUsed

		if !done {
			idc.procQ.enq(procToRun)
		}

	}

}
