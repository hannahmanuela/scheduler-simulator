package slasched

import (
	"fmt"
)

const (
	SCHED_QUANT    = 0.05
	THRESHOLD_QLEN = 5
)

type SchedulerType int

const (
	SHINJUKU SchedulerType = iota
	PS
)

type CoreSched struct {
	coreId         Tid
	totMem         Tmem
	coreQ          *Queue
	dispatcherConn chan *CoreMessages
	currTick       int
	machineId      Tid
	currSchedType  SchedulerType
}

func newCoreSched(dispatcherConn chan *CoreMessages, mid Tid, coreId Tid, schedType SchedulerType) *CoreSched {
	cs := &CoreSched{
		machineId:      mid,
		coreId:         coreId,
		totMem:         MAX_MEM_PER_CORE,
		coreQ:          newQueue(),
		dispatcherConn: dispatcherConn,
		currTick:       0,
		currSchedType:  schedType,
	}
	return cs
}

func (cs *CoreSched) String() string {
	str := fmt.Sprintf("{mem usage: %v, ", cs.memUsage())
	str += "q: \n"
	for _, p := range cs.coreQ.q {
		str += "    " + p.String() + "\n"
	}
	str += "}"
	return str
}

func (cs *CoreSched) printAllProcs() {
	for _, p := range cs.coreQ.q {
		fmt.Printf("current: %v, %v, %v, %v, %v, %v\n", cs.currTick, cs.machineId, cs.coreId, float64(p.procInternals.sla), float64(p.procInternals.actualComp), float64(p.procInternals.compDone))
	}
}

func (cs *CoreSched) memUsage() float64 {
	return float64(cs.memUsed()) / float64(cs.totMem)
}

func (cs *CoreSched) memUsed() Tmem {
	sum := Tmem(0)
	for _, p := range cs.coreQ.q {
		sum += p.memUsed()
	}
	return sum
}

func (cs *CoreSched) tick() {

	procToTicksMap := make(map[*Proc]TickBool, 0)
	switch cs.currSchedType {
	case SHINJUKU:
		procToTicksMap = cs.runProcsShinjuku()
	case PS:
		procToTicksMap = cs.runProcsPS()
	}

	if VERBOSE_SCHED_STATS {
		for proc, ticks := range procToTicksMap {
			fmt.Printf("sched: %v, %v, %v, %v, %v, %v, %v\n", cs.currTick, cs.machineId, cs.coreId, float64(proc.procInternals.sla), float64(proc.procInternals.compDone), float64(proc.ticksPassed), float64(ticks.tick))
		}
	}

	for _, currProc := range cs.coreQ.q {
		currProc.ticksPassed += 1
	}

	cs.currTick += 1
}

// right now only shinjuku cores call this
func (cs *CoreSched) potentiallyGetWork() {
	// if we don't have any, ask for work
	if cs.coreQ.qlen() < THRESHOLD_QLEN {
		cs.dispatcherConn <- &CoreMessages{NEED_WORK, nil, cs.currSchedType}
		msg := <-cs.dispatcherConn
		if msg.proc != nil {
			cs.coreQ.enq(msg.proc)
		}
	}
}

// do 1 tick of computation, spread across procs in q
func (cs *CoreSched) runProcsShinjuku() map[*Proc]TickBool {

	cs.potentiallyGetWork()

	ticksLeftToGive := Tftick(1)
	procToTicksMap := make(map[*Proc]TickBool, 0)

	for ticksLeftToGive-Tftick(0.0001) > 0.0 && cs.coreQ.qlen() > 0 {
		if VERBOSE_SCHEDULER {
			fmt.Printf("scheduling round: ticksLeftToGive is %v\n", ticksLeftToGive)
		}

		// get proc to run, which will be the one at the head of the q (earliest deadline first)
		procToRun := cs.getNextProc()
		ticksUsed, done := procToRun.runTillOutOrDone(SCHED_QUANT)
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
			if cs.memUsed() >= cs.totMem {
				fmt.Println("--> OUT OF MEMORY")
			}
		} else {
			// if the proc is done, update the ticksPassed to be exact for metrics etc
			// then update the pressure metric with that value
			procToRun.ticksPassed = procToRun.ticksPassed + (1 - ticksLeftToGive)
			// remove proc from q
			cs.coreQ.remove(procToRun)

			fmt.Printf("done: %v, %v, %v, %v, %v, %v\n", cs.currTick, cs.machineId, cs.coreId, float64(procToRun.procInternals.sla), float64(procToRun.ticksPassed), float64(procToRun.procInternals.actualComp))
		}

		cs.potentiallyGetWork()

	}

	return procToTicksMap
}

func (cs *CoreSched) runProcsPS() map[*Proc]TickBool {

	ticksLeftToGive := Tftick(1)
	procToTicksMap := make(map[*Proc]TickBool, 0)

	for ticksLeftToGive-Tftick(0.001) > 0.0 && cs.coreQ.qlen() > 0 {
		if VERBOSE_SCHEDULER {
			fmt.Printf("scheduling round: ticksLeftToGive is %v, so diff to 0.001 is %v\n", ticksLeftToGive, ticksLeftToGive-Tftick(0.001))
		}

		numProcs := cs.coreQ.qlen()
		newQ := make([]*Proc, 0)

		for _, procToRun := range cs.coreQ.q {

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
				if cs.memUsed() >= cs.totMem {
					fmt.Println("--> OUT OF MEMORY")
				}
				newQ = append(newQ, procToRun)
			} else {
				// if the proc is done, update the ticksPassed to be exact for metrics etc
				// then update the pressure metric with that value
				procToRun.ticksPassed = procToRun.ticksPassed + (1 - ticksLeftToGive)
			}
		}

		cs.coreQ.q = newQ

	}

	return procToTicksMap
}

func (cs *CoreSched) getNextProc() *Proc {
	nextProc := cs.coreQ.q[0]
	maxRatio := Tftick(0)

	for _, proc := range cs.coreQ.q {
		ratio := proc.ticksPassed / proc.getSla()
		if ratio > Tftick(maxRatio) {
			maxRatio = ratio
			nextProc = proc
		}
	}

	return nextProc
}
