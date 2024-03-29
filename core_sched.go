package slasched

import (
	"fmt"
)

const (
	THRESHOLD_QLEN       = 1
	TICK_SCHED_THRESHOLD = 0.00001 // given that 1 tick = 100ms (see website.go)
)

type CoreSched struct {
	q               *Queue
	machineConnSend chan *Message
	machineConnRecv chan *Message
	currTick        int
	machineId       Tid
	coreId          Tid
}

func newCoreSched(machineConnSend chan *Message, machineConnRecv chan *Message, mid Tid, cid Tid) *CoreSched {
	sd := &CoreSched{
		q:               newQueue(),
		machineConnSend: machineConnSend,
		machineConnRecv: machineConnRecv,
		currTick:        0,
		machineId:       mid,
		coreId:          cid,
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
	return float64(cs.memUsed()) / float64(MAX_MEM_PER_CORE)
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

func (cs *CoreSched) procsInRange(sla Tftick) int {
	slaBottom := getRangeBottomFromSLA(sla)
	numProcs := 0
	for _, p := range cs.q.getQ() {
		if getRangeBottomFromSLA(p.effectiveSla()) == slaBottom {
			numProcs += 1
		}
	}
	return numProcs
}

func (cs *CoreSched) tick() {
	cs.currTick += 1
	cs.runProcs()

}

func (cs *CoreSched) maxRatioTicksPassedToSla() float64 {
	max := 0.0
	for _, p := range cs.q.getQ() {
		if float64(p.ticksPassed/p.effectiveSla()) > max {
			max = float64(p.ticksPassed / p.effectiveSla())
		}
	}
	return max
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
			toWrite := fmt.Sprintf("%v, %v, %v, got proc from machine: %v \n", cs.currTick, cs.machineId, cs.coreId, msg.proc.String())
			logWrite(SCHED, toWrite)
			cs.q.enq(msg.proc)
		}
	}
}

// do 1 tick of computation
// run procs in q, asking for more if we don't have any or run out of them in the middle
// deq from q then run for an amount of time inversely prop to expectedComputationLeft
func (cs *CoreSched) runProcs() {
	cs.tryGetWork()

	ticksLeftToGive := Tftick(1)
	procToTicksMap := make(map[*Proc]TickBool, 0)

	toWrite := fmt.Sprintf("%v, %v, %v, curr q: %v \n", cs.currTick, cs.machineId, cs.coreId, cs.q.String())
	logWrite(SCHED, toWrite)

	for ticksLeftToGive-Tftick(TICK_SCHED_THRESHOLD) > 0.0 && cs.q.qlen() > 0 {

		// get proc to run, which will be the one at the head of the q (earliest deadline first)
		procToRun := cs.q.deq()
		ticksToGive := cs.allocTicksToProc(ticksLeftToGive, procToRun)
		ticksUsed, done, _ := procToRun.runTillOutOrDone(ticksToGive)
		ticksLeftToGive -= ticksUsed
		toWrite := fmt.Sprintf("%v, %v, %v running proc %v, gave %v ticks, used %v ticks\n", cs.currTick, cs.machineId, cs.coreId, procToRun.String(), ticksToGive.String(), ticksUsed.String())
		logWrite(SCHED, toWrite)

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
			if cs.memUsed() > MAX_MEM_PER_CORE {
				fmt.Println("--> OUT OF MEMORY")
				fmt.Printf("q: %v\n", cs.q.String())
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

	// if VERBOSE_SCHED_STATS {
	// 	for proc, ticks := range procToTicksMap {
	// 		if ticks.done {
	// 			toWrite := fmt.Sprintf("%v, %v, %v, %v, %v, %v, %v, 1\n", cs.currTick, cs.machineId, cs.coreId,
	// 				float64(proc.procInternals.sla), float64(proc.compUsed()), float64(proc.ticksPassed), float64(ticks.tick))
	// 			logWrite(SCHED, toWrite)
	// 		} else {
	// 			toWrite := fmt.Sprintf("%v, %v, %v, %v, %v, %v, %v, 0\n", cs.currTick, cs.machineId, cs.coreId,
	// 				float64(proc.procInternals.sla), float64(proc.compUsed()), float64(proc.ticksPassed), float64(ticks.tick))
	// 			logWrite(SCHED, toWrite)
	// 		}
	// 	}
	// }

}

func (cs *CoreSched) allocTicksToProc(ticksLeftToGive Tftick, procToRun *Proc) Tftick {

	// get values that allow us to inert the realtionsip between expectedCompLeft and ticks given
	// (because more time left should equal less ticks given)
	// TODO: is this the metric we want? or rather time left on sla?
	slaSum := procToRun.effectiveSla()
	for _, p := range cs.q.getQ() {
		slaSum += p.effectiveSla()
	}
	relativeNeedsSum := Tftick(slaSum / procToRun.effectiveSla())
	for _, p := range cs.q.getQ() {
		if p.effectiveSla() > 0 {
			relativeNeedsSum += slaSum / p.effectiveSla()
		}
	}

	allocatedTicks := ((slaSum / procToRun.effectiveSla()) / relativeNeedsSum) * ticksLeftToGive
	if allocatedTicks < 0 {
		fmt.Printf("ERROR -- allocated negative ticks. totalTimeLeft: %v, procToRun.expectedCompLeft() %v, relativeNeedsSum %v\n",
			slaSum, procToRun.effectiveSla(), relativeNeedsSum)
	}

	return allocatedTicks
}

func (cs *CoreSched) printAllProcs() {
	for _, p := range cs.q.getQ() {
		toWrite := fmt.Sprintf("%v, %v, %v, %v, %v, %v\n", cs.currTick, cs.machineId, cs.coreId,
			float64(p.procInternals.sla), float64(p.procInternals.actualComp), float64(p.compUsed()))
		logWrite(CURR_PROCS, toWrite)
	}
}

func (cs *CoreSched) tickAllProcs() {
	for _, p := range cs.q.getQ() {
		p.ticksPassed += 1
	}
}
