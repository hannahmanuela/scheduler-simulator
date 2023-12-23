package slasched

import (
	"fmt"
	"math"
)

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
	str := "{q: \n"
	for _, p := range sd.q.q {
		str += "    " + p.String() + "\n"
	}
	str += "}"
	return str
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

		ticksPerProc := sd.allocTicksToProc(ticksLeftToGive)
		newProcQ := make([]*Proc, 0)
		for _, currProc := range sd.q.q {
			ticksUsed, done := currProc.runTillOutOrDone(ticksPerProc[currProc])
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

// TODO: this is one of the big ones
func (sd *Schedd) allocTicksToProc(ticksLeftToGive Tftick) map[*Proc]Tftick {

	procToTicks := make(map[*Proc]Tftick, 0)

	if sd.q.qlen() == 1 {
		procToTicks[sd.q.q[0]] = Tftick(ticksLeftToGive)
		return procToTicks
	}

	// TODO: change this
	for _, currProc := range sd.q.q {
		allocatedTick := math.Min(float64(currProc.timeLeftOnSLA()), float64(ticksLeftToGive)/float64(sd.q.qlen()))
		fmt.Printf("giving %v ticks\n", allocatedTick)
		procToTicks[currProc] = Tftick(allocatedTick)
	}

	return procToTicks
}
