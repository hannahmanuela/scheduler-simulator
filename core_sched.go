package slasched

import (
	"fmt"
)

type CoreSched struct {
	totMem                 Tmem
	q                      *Queue
	avgDiffToSla           float64 // average left over budget (using an ewma with alpha value EWMA_ALPHA)
	numProcsKilledLastTick int
	ticksUnusedLastTick    Tftick
	machineSchedConn       chan *CoreMessages
	currTick               int
	machineId              Tid
	coreId                 Tid
}

func newCoreSched(machineSchedConn chan *CoreMessages, mid Tid, cid Tid) *CoreSched {
	sd := &CoreSched{
		totMem:                 MAX_MEM_PER_CORE,
		q:                      newQueue(),
		avgDiffToSla:           0,
		numProcsKilledLastTick: 0,
		machineSchedConn:       machineSchedConn,
		ticksUnusedLastTick:    0,
		currTick:               0,
		machineId:              mid,
		coreId:                 cid,
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

func (cd *CoreSched) checkIfGotMessage() {
	select {
	case msg := <-cd.machineSchedConn:
		fmt.Println("looked for messages on core and had one")
		switch msg.msgType {
		case PUSH_PROC:
			cd.q.enq(msg.proc)
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
	for _, p := range cs.q.getQ() {
		sum += p.memUsed()
	}
	return sum
}

// returns the amount of ticks of projected work that the scheduler has before it would get to the given deadline
func (cs *CoreSched) getTicksAhead(deadline Tftick) Tftick {
	sum := Tftick(0)
	for _, p := range cs.q.getQ() {
		if p.timeShouldBeDone < deadline {
			sum += p.expectedCompLeft()
		}
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

// do 1 tick of computation, spread across procs in q, and across different cores
// TODO: the way I ask for work right now is stupid I should batch things
func (cs *CoreSched) runProcs() {

	cs.checkIfGotMessage()

	if cs.q.qlen() < THRESHOLD_QLEN {
		cs.machineSchedConn <- &CoreMessages{NEED_WORK, nil}
		msg := <-cs.machineSchedConn
		if msg.proc != nil {
			cs.q.enq(msg.proc)
		}
	}

	ticksLeftToGive := Tftick(1)
	procToTicksMap := make(map[*Proc]TickBool, 0)

	for ticksLeftToGive-Tftick(0.001) > 0.0 && cs.q.qlen() > 0 {
		if VERBOSE_SCHEDULER {
			fmt.Printf("scheduling round: ticksLeftToGive is %v, so diff to 0.001 is %v\n", ticksLeftToGive, ticksLeftToGive-Tftick(0.001))
		}

		// get proc to run, which will be the one at the head of the q (earliest deadline first)
		procToRun := cs.q.deq()
		ticksToGive := cs.allocTicksToProc(ticksLeftToGive, procToRun)
		ticksUsed, done, _ := procToRun.runTillOutOrDone(ticksToGive)
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
				if VERBOSE_SCHEDULER {
					fmt.Println("--> KILLING")
				}
				// TODO: handle killing
				// cs.kill()
			}
			// add proc back into queue
			cs.q.enq(procToRun)
		} else {
			// if the proc is done, update the ticksPassed to be exact for metrics etc
			procToRun.ticksPassed = procToRun.ticksPassed + (1 - ticksLeftToGive)
			// don't need to wait if we are just telling it a proc is done
			cs.machineSchedConn <- &CoreMessages{PROC_DONE_CORE, procToRun}
		}

		if cs.q.qlen() < THRESHOLD_QLEN {
			cs.machineSchedConn <- &CoreMessages{NEED_WORK, nil}
			msg := <-cs.machineSchedConn
			if msg.proc != nil {
				cs.q.enq(msg.proc)
			}
		}

	}

	cs.ticksUnusedLastTick = ticksLeftToGive

	if VERBOSE_SCHED_STATS {
		for proc, ticks := range procToTicksMap {
			if ticks.done {
				fmt.Printf("sched: %v, %v, %v, %v, %v, %v, %v, 1\n", cs.currTick, cs.machineId, cs.coreId,
					float64(proc.procInternals.sla), float64(proc.procInternals.compDone), float64(proc.ticksPassed), float64(ticks.tick))
			} else {
				fmt.Printf("sched: %v, %v, %v, %v, %v, %v, %v, 0\n", cs.currTick, cs.machineId, cs.coreId,
					float64(proc.procInternals.sla), float64(proc.procInternals.compDone), float64(proc.ticksPassed), float64(ticks.tick))
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
	// also find out if there are procs over the SLA, and if yes how many
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
	if VERBOSE_SCHEDULER {
		fmt.Printf("giving %v ticks \n", allocatedTicks)
	}

	return allocatedTicks
}

// func (cs *CoreSched) kill() {
// 	// TODO: this should probbaly have a global write lock on q
// 	// TODO: and should move to q?

// 	// sort by killable score :D
// 	currQ := cs.q.getQ()
// 	sort.Slice(currQ, func(i, j int) bool {
// 		return currQ[i].killableScore() > currQ[j].killableScore()
// 	})

// 	memOver := cs.memUsed() - MAX_MEM_PER_CORE
// 	memCut := Tmem(0)

// 	// this threshold is kinda arbitrary
// 	for memCut < memOver {
// 		killed := cs.q.q[0]
// 		cs.q.q = cs.q.q[1:]
// 		memCut += killed.memUsed()
// 		killed.migrated = true
// 		// var wg sync.WaitGroup
// 		// wg.Add(1)
// 		cs.machineSchedConn <- &MachineMessages{PROC_KILLED, killed, nil}
// 		// wg.Wait()
// 	}
// }
