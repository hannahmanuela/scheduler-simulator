package slasched

import (
	"fmt"
	"math"
)

const (
	PUSH_SLA_THRESHOLD     = 2   // 1 tick = 100 ms ==> 5 ms (see website.go)
	PUSH_RATIO_THRESHOLD   = 0.3 // if a proc has waited in the machine's q for longer than this percentage of its SLA, push it to a core
	ACTIVEQ_PUSH_THRESHOLD = 0.5 // if the active q has less ticks than this in it, push new procs into the active q right away

	TICK_SCHED_THRESHOLD = 0.00001 // amount of ticks after which I stop scheduling; given that 1 tick = 100ms (see website.go)
)

type Sched struct {
	machineId  Tid
	holdQ      *Queue
	activeQ    *Queue
	lbConnSend chan *Message // channel to send messages to LB
	lbConnRecv chan *Message // channel this machine recevies messages on from the LB
	currTick   int
}

func newSched(lbConnSend chan *Message, lbConnRecv chan *Message, mid Tid) *Sched {
	sd := &Sched{
		machineId:  mid,
		holdQ:      newQueue(),
		activeQ:    newQueue(),
		lbConnSend: lbConnSend,
		lbConnRecv: lbConnRecv,
		currTick:   0,
	}
	go sd.runLBConn()
	return sd
}

func (sd *Sched) String() string {
	return fmt.Sprintf("machine scheduler: %v", sd.machineId)
}

func (sd *Sched) tick() {
	sd.currTick += 1

	// push procs onto core if they have waited for long enough
	newHoldQ := []*Proc{}
	for _, p := range sd.holdQ.getQ() {
		if (p.ticksPassed / p.effectiveSla()) >= PUSH_RATIO_THRESHOLD {
			sd.activeQ.q = append(sd.activeQ.q, p)
		} else {
			newHoldQ = append(newHoldQ, p)
		}
	}
	sd.holdQ.q = newHoldQ

	sd.simulateRunProcs()
}

func (sd *Sched) printAllProcs() {
	for _, p := range sd.holdQ.getQ() {
		toWrite := fmt.Sprintf("%v, %v, 0, %v, %v, %v\n", sd.currTick, sd.machineId,
			float64(p.procInternals.sla), float64(p.procInternals.actualComp), float64(p.compUsed()))
		logWrite(CURR_PROCS, toWrite)
	}
	for _, p := range sd.activeQ.getQ() {
		toWrite := fmt.Sprintf("%v, %v, 1, %v, %v, %v\n", sd.currTick, sd.machineId,
			float64(p.procInternals.sla), float64(p.procInternals.actualComp), float64(p.compUsed()))
		logWrite(CURR_PROCS, toWrite)
	}
}

func (sd *Sched) tickAllProcs() {
	for _, p := range append(sd.holdQ.getQ(), sd.activeQ.getQ()...) {
		p.ticksPassed += 1
	}
}

func (sd *Sched) runLBConn() {

	// listen to messages
	for {
		msg := <-sd.lbConnRecv
		switch msg.msgType {
		case LB_M_PLACE_PROC:
			if msg.proc.effectiveSla() < PUSH_SLA_THRESHOLD || sd.expectedCompInActiveQ() < ACTIVEQ_PUSH_THRESHOLD {
				sd.activeQ.enq(msg.proc)
			} else {
				sd.holdQ.enq(msg.proc)
			}
			msg.wg.Done()
		}
	}

}

func (sd *Sched) memUsed() Tmem {
	memUsed := Tmem(0)
	for _, p := range sd.activeQ.getQ() {
		memUsed += p.memUsed()
	}
	return memUsed
}

func (sd *Sched) memFree() float64 {
	return MAX_MEM_PER_MACHINE - float64(sd.memUsed())
}

func (sd *Sched) memUsage() float64 {
	return float64(sd.memUsed()) / float64(MAX_MEM_PER_MACHINE)
}

func (sd *Sched) ticksInQ() float64 {
	totalTicks := Tftick(0)
	for _, p := range append(sd.holdQ.getQ(), sd.activeQ.getQ()...) {
		totalTicks += p.effectiveSla()
	}
	return float64(totalTicks)
}

func (sd *Sched) expectedCompInQ() float64 {
	totalTicks := Tftick(0)
	for _, p := range append(sd.holdQ.getQ(), sd.activeQ.getQ()...) {
		totalTicks += p.profilingExpectedCompLeft()
	}
	return float64(totalTicks)
}

// expected based on profiling info
func (sd *Sched) expectedCompInActiveQ() float64 {
	totalTicks := Tftick(0)
	for _, p := range sd.activeQ.getQ() {
		totalTicks += p.profilingExpectedCompLeft()
	}
	return float64(totalTicks)
}

func (sd *Sched) procsInRange(sla Tftick) int {
	slaBottom := getRangeBottomFromSLA(sla)
	numProcs := 0
	for _, p := range append(sd.activeQ.getQ(), sd.holdQ.getQ()...) {
		if getRangeBottomFromSLA(p.effectiveSla()) == slaBottom {
			numProcs += 1
		}
	}
	return numProcs
}

func (sd *Sched) maxRatioTicksPassedToSla() float64 {
	max := 0.0
	for _, p := range append(sd.activeQ.getQ(), sd.holdQ.getQ()...) {
		if float64(p.ticksPassed/p.effectiveSla()) > max {
			max = float64(p.ticksPassed / p.effectiveSla())
		}
	}
	return max
}

// do numCores ticks of computation (only on procs in the activeQ)
func (sd *Sched) simulateRunProcs() {

	if VERBOSE_MACHINE_USAGE_STATS {
		toWrite := fmt.Sprintf("%v, %v, %.2f, %.2f, %v, %v", sd.currTick, sd.machineId,
			sd.maxRatioTicksPassedToSla(), sd.memUsage(), sd.activeQ.qlen(), sd.ticksInQ()) //, cs.q.String()
		logWrite(USAGE, toWrite)
	}

	ticksLeftToGive := Tftick(1)

	toWrite := fmt.Sprintf("%v, %v, curr q ACTIVE: %v, curr q HOLD: %v \n", sd.currTick, sd.machineId, sd.activeQ.String(), sd.holdQ.String())
	logWrite(SCHED, toWrite)

	for ticksLeftToGive-Tftick(TICK_SCHED_THRESHOLD) > 0.0 && sd.activeQ.qlen() > 0 {

		ticksPerProc := sd.allocTicksToProcs(ticksLeftToGive)
		newQ := []*Proc{}

		for _, procToRun := range sd.activeQ.getQ() {
			ticksToGive := ticksPerProc[procToRun]
			ticksUsed, done := procToRun.runTillOutOrDone(ticksToGive)
			ticksLeftToGive -= ticksUsed
			toWrite := fmt.Sprintf("%v, %v, running proc %v, gave %v ticks, used %v ticks\n", sd.currTick, sd.machineId, procToRun.String(), ticksToGive.String(), ticksUsed.String())
			logWrite(SCHED, toWrite)

			if !done {
				// check if the memroy used by the proc sent us over the edge (and if yes, kill as needed)
				if sd.memUsed() > MAX_MEM_PER_MACHINE {
					fmt.Println("--> OUT OF MEMORY")
					fmt.Printf("q: %v\n", sd.activeQ.String())
				}
				// add proc back into queue
				newQ = append(newQ, procToRun)
			} else {
				// if the proc is done, update the ticksPassed to be exact for metrics etc
				procToRun.ticksPassed = procToRun.ticksPassed + (1 - ticksLeftToGive)
				// don't need to wait if we are just telling it a proc is done
				sd.lbConnSend <- &Message{sd.machineId, M_LB_PROC_DONE, procToRun, nil}
			}
		}
		sd.activeQ.q = newQ

		// if activeQ empty, steal from holdQ if possible
		if sd.activeQ.qlen() == 0 && sd.holdQ.qlen() > 0 {
			procToMove := sd.holdQ.deq()
			sd.activeQ.enq(procToMove)
		}
	}

	// this is dumb but make accounting for util easier
	if ticksLeftToGive < 0.00002 {
		ticksLeftToGive = 0
	}
	if VERBOSE_MACHINE_USAGE_STATS {
		toWrite := fmt.Sprintf(", %v\n", float64(math.Copysign(float64(ticksLeftToGive), 1)))
		logWrite(USAGE, toWrite)
	}
}

func (sd *Sched) allocTicksToProcs(ticksLeftToGive Tftick) map[*Proc]Tftick {

	// get values that allow us to inert the realtionsip between expectedCompLeft and ticks given
	// (because more time left should equal less ticks given)
	procToTicks := make(map[*Proc]Tftick, 0)

	totalTimeLeft := Tftick(0)
	for _, p := range sd.activeQ.getQ() {
		totalTimeLeft += p.effectiveSla()
		// totalTimeLeft += p.expectedCompLeft()

	}

	relativeNeedsSum := Tftick(0)
	for _, p := range sd.activeQ.getQ() {
		if p.effectiveSla() > 0 {
			relativeNeedsSum += totalTimeLeft / p.effectiveSla()
			// relativeNeedsSum += totalTimeLeft / p.expectedCompLeft()
		}
	}

	ticksGiven := Tftick(0)
	for _, p := range sd.activeQ.getQ() {
		allocatedTicks := ((totalTimeLeft / p.effectiveSla()) / relativeNeedsSum) * ticksLeftToGive
		// allocatedTicks := ((totalTimeLeft / p.expectedCompLeft()) / relativeNeedsSum) * ticksLeftToGive
		// toWrite := fmt.Sprintf("%v, %v, %v allocating to proc %v, gave %v ticks, because of %v totalTimeLeft and %v ratio btw total and procs timeleft \n", cs.currTick, cs.machineId, cs.coreId, currProc.String(), allocatedTicks, totalTimeLeft, max(currProc.timeLeftOnSLA(), 0.00001))
		// logWrite(SCHED, toWrite)
		procToTicks[p] = allocatedTicks
		ticksGiven += allocatedTicks
	}

	return procToTicks
}
