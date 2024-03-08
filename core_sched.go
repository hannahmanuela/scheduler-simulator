package slasched

import (
	"fmt"
)

const (
	SCHED_QUANT    = 0.05
	THRESHOLD_QLEN = 2
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
}

func newCoreSched(dispatcherConn chan *CoreMessages, mid Tid, coreId Tid) *CoreSched {
	cs := &CoreSched{
		machineId:      mid,
		coreId:         coreId,
		totMem:         MAX_MEM_PER_CORE,
		coreQ:          newQueue(),
		dispatcherConn: dispatcherConn,
		currTick:       0,
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
		fmt.Printf("current: %v, %v, %v, %v, %v\n", cs.currTick, cs.machineId, float64(p.procInternals.sla), float64(p.procInternals.actualComp), float64(p.procInternals.compDone))
	}
}

func (cs *CoreSched) checkIfGotMessage() {
	select {
	case msg := <-cs.dispatcherConn:
		fmt.Println("looked for messages on core and had one")
		switch msg.msgType {
		case PUSH_PROC:
			cs.coreQ.enq(msg.proc)
		}
	default:
		return
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
	cs.currTick += 1
	if len(cs.coreQ.q) == 0 {
		return
	}
	cs.runProcs()
	for _, currProc := range cs.coreQ.q {
		currProc.ticksPassed += 1
	}
}

func (cs *CoreSched) potentiallyGetWork() {
	// see if core was sent work
	cs.checkIfGotMessage()

	// if we don't have any, ask for work
	if cs.coreQ.qlen() < THRESHOLD_QLEN {
		cs.dispatcherConn <- &CoreMessages{NEED_WORK, nil}
		fmt.Println("core asking for work")
		msg := <-cs.dispatcherConn
		if msg.proc != nil {
			cs.coreQ.enq(msg.proc)
		}
	}
}

// do 1 tick of computation, spread across procs in q, and across different cores
func (cs *CoreSched) runProcs() {

	cs.potentiallyGetWork()

	ticksLeftToGive := Tftick(1)
	procToTicksMap := make(map[*Proc]TickBool, 0)

	for ticksLeftToGive-Tftick(0.0001) > 0.0 && cs.coreQ.qlen() > 0 {
		if VERBOSE_SCHEDULER {
			fmt.Printf("scheduling round: ticksLeftToGive is %v, so diff to 0.001 is %v\n", ticksLeftToGive, ticksLeftToGive-Tftick(0.001))
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

			fmt.Printf("done: %v, %v, %v, %v, %v, %v\n", cs.currTick, cs.machineId, procToRun.procInternals.procType, float64(procToRun.procInternals.sla), float64(procToRun.ticksPassed), float64(procToRun.procInternals.actualComp))
		}

		cs.potentiallyGetWork()

	}

	if VERBOSE_SCHED_STATS {
		for proc, ticks := range procToTicksMap {
			if ticks.done {
				fmt.Printf("sched: %v, %v, %v, %v, %v, %v, 1\n", cs.currTick, cs.machineId, float64(proc.procInternals.sla), float64(proc.procInternals.compDone), float64(proc.ticksPassed), float64(ticks.tick))
			} else {
				fmt.Printf("sched: %v, %v, %v, %v, %v, %v, 0\n", cs.currTick, cs.machineId, float64(proc.procInternals.sla), float64(proc.procInternals.compDone), float64(proc.ticksPassed), float64(ticks.tick))
			}
		}
	}

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
