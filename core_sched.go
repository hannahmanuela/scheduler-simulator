package slasched

import (
	"fmt"
)

const (
	THRESHOLD_QLEN       = 1
	TICK_SCHED_THRESHOLD = 0.00001 // given that 1 tick = 100ms (see website.go)
)

type CoreSched struct {
	totMem              Tmem
	q                   *Queue
	ticksUnusedLastTick Tftick
	machineConnSend     chan *Message
	machineConnRecv     chan *Message
	currTick            int
	machineId           Tid
	coreId              Tid
}

func newCoreSched(machineConnSend chan *Message, machineConnRecv chan *Message, mid Tid, cid Tid) *CoreSched {
	sd := &CoreSched{
		totMem:              MAX_MEM_PER_CORE,
		q:                   newQueue(),
		machineConnSend:     machineConnSend,
		machineConnRecv:     machineConnRecv,
		ticksUnusedLastTick: 0,
		currTick:            0,
		machineId:           mid,
		coreId:              cid,
	}
	return sd
}

func (cs *CoreSched) String() string {
	str := fmt.Sprintf("{mem usage: %v, ", cs.memUsage())
	str += "q: \n"
	for _, p := range cs.q.getQ() {
		str += "    " + p.String() + "\n"
	}
	str += "}"
	return str
}

func (cs *CoreSched) memUsage() float64 {
	return float64(cs.memUsed()) / float64(cs.totMem)
}

func (cs *CoreSched) memUsed() Tmem {
	sum := Tmem(0)
	for _, p := range cs.q.getQ() {
		sum += p.memUsed()
	}
	return sum
}

func (cs *CoreSched) ticksInQ() Tftick {
	sum := Tftick(0)
	for _, p := range cs.q.getQ() {
		sum += p.expectedCompLeft()
	}
	return sum
}

func (cs *CoreSched) tick() {
	cs.currTick += 1
	cs.runProcs()
	for _, currProc := range cs.q.getQ() {
		currProc.ticksPassed += 1
	}
}

type TickBool struct {
	tick Tftick
	done bool
}

func (cs *CoreSched) tryGetWork() {
	if cs.q.qlen() < THRESHOLD_QLEN {
		cs.machineConnSend <- &Message{cs.coreId, C_M_NEED_WORK, nil, nil}
		msg := <-cs.machineConnRecv
		if msg.proc != nil {
			cs.q.enq(msg.proc)
		}
	}
}

// do 1 tick of computation
// run procs in q, asking for more if we don't have any or run out of them in the middle
// deq from q then run for an amount of time inversely prop to expectedComputationLeft
// TODO: the way I ask for work right now is stupid I should batch things?
func (cs *CoreSched) runProcs() {
	cs.tryGetWork()

	ticksLeftToGive := Tftick(1)
	procToTicksMap := make(map[*Proc]TickBool, 0)

	for ticksLeftToGive-Tftick(TICK_SCHED_THRESHOLD) > 0.0 && cs.q.qlen() > 0 {

		// get proc to run, which will be the one at the head of the q (earliest deadline first)
		procToRun := cs.q.deq()
		ticksToGive := cs.allocTicksToProc(ticksLeftToGive, procToRun)
		ticksUsed, done, _ := procToRun.runTillOutOrDone(ticksToGive)
		ticksLeftToGive -= ticksUsed

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
			// add proc back into queue
			cs.q.enq(procToRun)
		} else {
			// if the proc is done, update the ticksPassed to be exact for metrics etc
			procToRun.ticksPassed = procToRun.ticksPassed + (1 - ticksLeftToGive)
			// don't need to wait if we are just telling it a proc is done
			cs.machineConnSend <- &Message{cs.coreId, C_M_PROC_DONE, procToRun, nil}
		}

		cs.tryGetWork()

	}

	cs.ticksUnusedLastTick = ticksLeftToGive

	if VERBOSE_SCHED_STATS {
		for proc, ticks := range procToTicksMap {
			if ticks.done {
				toWrite := fmt.Sprintf("%v, %v, %v, %v, %v, %v, %v, 1\n", cs.currTick, cs.machineId, cs.coreId,
					float64(proc.procInternals.sla), float64(proc.compUsed()), float64(proc.ticksPassed), float64(ticks.tick))
				logWrite(SCHED, toWrite)
			} else {
				toWrite := fmt.Sprintf("%v, %v, %v, %v, %v, %v, %v, 0\n", cs.currTick, cs.machineId, cs.coreId,
					float64(proc.procInternals.sla), float64(proc.compUsed()), float64(proc.ticksPassed), float64(ticks.tick))
				logWrite(SCHED, toWrite)
			}
		}
	}

}

// allocates ticks
// inversely proportional to how much expected computation the proc has left
// if there are procs that are over, will (for now, equally) spread all ticks between them
func (cs *CoreSched) allocTicksToProc(ticksLeftToGive Tftick, procToRun *Proc) Tftick {

	// get values that allow us to inert the realtionsip between expectedCompLeft and ticks given
	// (because more time left should equal less ticks given)
	// TODO: is this the metric we want? or rather time left on sla?
	totalTimeLeft := procToRun.expectedCompLeft()
	for _, p := range cs.q.getQ() {
		if p.expectedCompLeft() <= 0 {
			fmt.Printf("ERROR -- somehow a proc has negative time left -- shouldn't it have been replenished?\n")
		} else {
			totalTimeLeft += p.expectedCompLeft()
		}
	}
	relativeNeedsSum := Tftick(totalTimeLeft / procToRun.expectedCompLeft())
	for _, p := range cs.q.getQ() {
		if p.expectedCompLeft() > 0 {
			relativeNeedsSum += totalTimeLeft / p.expectedCompLeft()
		}
	}

	allocatedTicks := ((totalTimeLeft / procToRun.expectedCompLeft()) / relativeNeedsSum) * ticksLeftToGive
	if allocatedTicks < 0 {
		fmt.Printf("ERROR -- allocated negative ticks. totalTimeLeft: %v, procToRun.expectedCompLeft() %v, relativeNeedsSum %v\n",
			totalTimeLeft, procToRun.expectedCompLeft(), relativeNeedsSum)
	}

	return allocatedTicks
}
