package slasched

import (
	"fmt"
	"sort"
)

const (
	EWMA_ALPHA = 0.2
)

type Sched struct {
	totMem                 Tmem
	q                      *Queue
	avgDiffToSla           float64 // average left over budget (using an ewma with alpha value EWMA_ALPHA)
	numProcsKilledLastTick int
	ticksUnusedLastTick    Tftick
	lbConn                 chan *MachineMessages
	currTick               int
	machineId              Tmid
}

func newSched(lbConn chan *MachineMessages, mid Tmid) *Sched {
	sd := &Sched{
		totMem:                 MAX_MEM,
		q:                      newQueue(),
		avgDiffToSla:           0,
		numProcsKilledLastTick: 0,
		ticksUnusedLastTick:    0,
		lbConn:                 lbConn,
		currTick:               0,
		machineId:              mid,
	}
	return sd
}

func (sd *Sched) String() string {
	str := fmt.Sprintf("{mem usage: %v, ", sd.memUsage())
	str += "q: \n"
	for _, p := range sd.q.q {
		str += "    " + p.String() + "\n"
	}
	str += "}"
	return str
}

func (sd *Sched) memUsage() float64 {
	return float64(sd.memUsed()) / float64(sd.totMem)
}

func (sd *Sched) memUsed() Tmem {
	sum := Tmem(0)
	for _, p := range sd.q.q {
		sum += p.memUsed()
	}
	return sum
}

// returns the amount of ticks of projected work that the scheduler has before it would get to the given deadline
func (sd *Sched) getTicksAhead(deadline Tftick) Tftick {
	sum := Tftick(0)
	for _, p := range sd.q.q {
		if p.timeShouldBeDone < deadline {
			sum += p.expectedCompLeft()
		}
	}
	return sum
}

func (sd *Sched) tick() {
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

type TickBool struct {
	tick Tftick
	done bool
}

// do 1 tick of computation, spread across procs in q, and across different cores
func (sd *Sched) runProcs() {
	ticksLeftToGive := Tftick(1)
	procToTicksMap := make(map[*Proc]TickBool, 0)

	for ticksLeftToGive-Tftick(0.001) > 0.0 && sd.q.qlen() > 0 {
		if VERBOSE_SCHEDULER {
			fmt.Printf("scheduling round: ticksLeftToGive is %v, so diff to 0.001 is %v\n", ticksLeftToGive, ticksLeftToGive-Tftick(0.001))
		}

		// get proc to run, which will be the one at the head of the q (earliest deadline first)
		procToRun := sd.q.deq()
		ticksUsed, done := procToRun.runTillOutOrDone(ticksLeftToGive)
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
				sd.kill()
			}
			// add proc back into queue
			sd.q.enq(procToRun)
		} else {
			// if the proc is done, update the ticksPassed to be exact for metrics etc
			// then update the pressure metric with that value
			procToRun.ticksPassed = procToRun.ticksPassed + (1 - ticksLeftToGive)
			sd.updateAvgDiffToSLA(procToRun.timeLeftOnSLA())
			// don't need to wait if we are just telling it a proc is done
			sd.lbConn <- &MachineMessages{PROC_DONE, procToRun, nil}
		}

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

// currently just using EWMA, in both cases (went over SLA and was still under)
func (sd *Sched) updateAvgDiffToSLA(diffToSLA Tftick) {
	if VERBOSE_SCHEDULER {
		fmt.Printf("updating pressure given diff: %v \n", diffToSLA)
	}
	sd.avgDiffToSla = EWMA_ALPHA*float64(diffToSLA) + (1-EWMA_ALPHA)*sd.avgDiffToSla
}

func (sd *Sched) kill() {

	// sort by killable score :D
	sort.Slice(sd.q.q, func(i, j int) bool {
		return sd.q.q[i].killableScore() > sd.q.q[j].killableScore()
	})

	memOver := sd.memUsed() - MAX_MEM
	memCut := Tmem(0)

	// this threshold is kinda arbitrary
	for memCut < memOver {
		killed := sd.q.q[0]
		sd.q.q = sd.q.q[1:]
		memCut += killed.memUsed()
		killed.migrated = true
		// var wg sync.WaitGroup
		// wg.Add(1)
		sd.lbConn <- &MachineMessages{PROC_KILLED, killed, nil}
		sd.numProcsKilledLastTick += 1
		// wg.Wait()
	}
}
