package slasched

import (
	"fmt"
)

type PSSched struct {
	totMem                 Tmem
	q                      *Queue
	numProcsKilledLastTick int
	ticksUnusedLastTick    Tftick
	lbConn                 chan *MachineMessages
	currTick               int
	machineId              Tid
}

func newPSSched(lbConn chan *MachineMessages, mid Tid) *PSSched {
	sd := &PSSched{
		totMem:                 MAX_MEM_PER_CORE,
		q:                      newQueue(),
		numProcsKilledLastTick: 0,
		ticksUnusedLastTick:    0,
		lbConn:                 lbConn,
		currTick:               0,
		machineId:              mid,
	}
	return sd
}

func (sd *PSSched) String() string {
	str := fmt.Sprintf("{mem usage: %v, ", sd.memUsage())
	str += "q: \n"
	for _, p := range sd.q.q {
		str += "    " + p.String() + "\n"
	}
	str += "}"
	return str
}

func (sd *PSSched) getQ() *Queue {
	return sd.q
}

func (sd *PSSched) memUsage() float64 {
	return float64(sd.memUsed()) / float64(sd.totMem)
}

func (sd *PSSched) memUsed() Tmem {
	sum := Tmem(0)
	for _, p := range sd.q.q {
		sum += p.memUsed()
	}
	return sum
}

// returns the amount of ticks of projected work that the scheduler has before it would get to the given deadline
func (sd *PSSched) getTicksAhead(deadline Tftick) Tftick {
	sum := Tftick(0)
	for _, p := range sd.q.q {
		if p.timeShouldBeDone < deadline {
			sum += p.expectedCompLeft()
		}
	}
	return sum
}

func (sd *PSSched) tick() {
	sd.currTick += 1
	sd.numProcsKilledLastTick = 0
	sd.ticksUnusedLastTick = 0
	if len(sd.q.q) == 0 {
		return
	}
	sd.runProcs()
	for _, currProc := range sd.q.q {
		currProc.ticksPassed += 1
	}
}

// do 1 tick of computation, spread across procs in q, and across different cores
func (sd *PSSched) runProcs() {
	ticksLeftToGive := Tftick(1)
	procToTicksMap := make(map[*Proc]TickBool, 0)

	for ticksLeftToGive-Tftick(0.001) > 0.0 && sd.q.qlen() > 0 {
		if VERBOSE_SCHEDULER {
			fmt.Printf("scheduling round: ticksLeftToGive is %v, so diff to 0.001 is %v\n", ticksLeftToGive, ticksLeftToGive-Tftick(0.001))
		}

		numProcs := sd.q.qlen()
		newQ := make([]*Proc, 0)

		for _, procToRun := range sd.q.q {

			ticksUsed, done := procToRun.runTillOutOrDone(ticksLeftToGive / Tftick(numProcs))
			ticksLeftToGive -= ticksUsed
			if VERBOSE_SCHEDULER {
				fmt.Printf("used %v ticks\n", ticksUsed)
			}

			// add ticks used to the tick map
			if val, ok := procToTicksMap[procToRun]; ok {
				val.tick += ticksUsed
				val.done = done
				procToTicksMap[procToRun] = val
			} else {
				procToTicksMap[procToRun] = TickBool{ticksUsed, done}
			}

			if !done {
				// check if the memroy used by the proc sent us over the edge (and if yes, kill as needed)
				if sd.memUsed() >= sd.totMem {
					if VERBOSE_SCHEDULER {
						fmt.Println("--> KILLING")
					}
				}
				newQ = append(newQ, procToRun)
			} else {
				// if the proc is done, update the ticksPassed to be exact for metrics etc
				// then update the pressure metric with that value
				procToRun.ticksPassed = procToRun.ticksPassed + (1 - ticksLeftToGive)
			}
		}

		sd.q.q = newQ

	}

	sd.ticksUnusedLastTick = ticksLeftToGive

	if VERBOSE_SCHED_STATS {
		for proc, ticks := range procToTicksMap {
			if ticks.done {
				fmt.Printf("sched: %v, %v, %v, %v, %v, %v, 1\n", sd.currTick, sd.machineId, float64(proc.procInternals.sla), float64(proc.procInternals.compDone), float64(proc.ticksPassed), float64(ticks.tick))
			} else {
				fmt.Printf("sched: %v, %v, %v, %v, %v, %v, 0\n", sd.currTick, sd.machineId, float64(proc.procInternals.sla), float64(proc.procInternals.compDone), float64(proc.ticksPassed), float64(ticks.tick))
			}
		}
	}

}
