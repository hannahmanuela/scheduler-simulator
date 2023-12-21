package main

import "fmt"

type Schedd struct {
	totMem Tmem
	q      *Queue
}

func newSchedd() *Schedd {
	sd := &Schedd{
		totMem: MAX_MEM,
		q:      newQueue(),
	}
	return sd
}

func (sd *Schedd) String() string {
	return fmt.Sprintf("{totMem %d memUsed %d q %v}", sd.totMem, sd.memUsed(), sd.q)
}

func (sd *Schedd) memUsed() Tmem {
	m := Tmem(0)
	for _, p := range sd.q.q {
		m += p.memUsed
	}
	return m
}

func (sd *Schedd) tick() {
	if len(sd.q.q) == 0 {
		return
	}
	sd.runProcs()
	for _, currProc := range sd.q.q {
		currProc.ticksPassed += 1
	}
}

// do 1 tick of computation, spread across procs in q
func (sd *Schedd) runProcs() {
	ticksLeftToGive := Tftick(1)
	for ticksLeftToGive > 0 && sd.q.qlen() > 0 {
		newProcQ := make([]*Proc, 0)
		for _, currProc := range sd.q.q {
			ticksForCurrProc := sd.q.allocTicksToProc(ticksLeftToGive, currProc)
			ticksUsed, done := currProc.runTillOutOrDone(ticksForCurrProc)
			if !done {
				newProcQ = append(newProcQ, currProc)
			}
			ticksLeftToGive -= ticksUsed
		}
		sd.q.q = newProcQ
		if sd.q.qlen() > 0 {
			if ticksLeftToGive > 0.001 {
				fmt.Printf("another round of scheduling %v\n", ticksLeftToGive)
			} else {
				ticksLeftToGive = Tftick(0)
			}
		}
	}
}

// TODO: this
func (q *Queue) allocTicksToProc(ticksLeftToGive Tftick, proc *Proc) Tftick {
	return 0
}
