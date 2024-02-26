package slasched

func tickTheoreticalGlobalEDF(q *Queue) {
	ticksLeftToGive := Tftick(1)
	for ticksLeftToGive-Tftick(0.001) > 0.0 && q.qlen() > 0 {
		topProc := q.deq()
		ticksUsed, done, _ := topProc.runTillOutOrDone(ticksLeftToGive)
		ticksLeftToGive -= ticksUsed
		if !done {
			q.enq(topProc)
		}
	}
}
