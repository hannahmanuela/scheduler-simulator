package slasched

import (
	"fmt"
	"math"
)

const (
	TICK_SCHED_THRESHOLD = 0.00001 // amount of ticks after which I stop scheduling; given that 1 tick = 100ms (see website.go)
)

type Sched struct {
	machineId  Tid
	activeQ    *Queue
	blockedQ   *Queue
	lbConnSend chan *Message // channel to send messages to LB
	lbConnRecv chan *Message // channel this machine recevies messages on from the LB
	currTick   int
}

func newSched(lbConnSend chan *Message, lbConnRecv chan *Message, mid Tid) *Sched {
	sd := &Sched{
		machineId:  mid,
		activeQ:    newQueue(),
		blockedQ:   newQueue(),
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

	sd.simulateRunProcs()
}

func (sd *Sched) printAllProcs() {

	for _, p := range append(sd.activeQ.getQ(), sd.blockedQ.getQ()...) {
		toWrite := fmt.Sprintf("%v, %v, 1, %v, %v, %v\n", sd.currTick, sd.machineId,
			float64(p.procInternals.sla), float64(p.procInternals.actualComp), float64(p.compUsed()))
		logWrite(CURR_PROCS, toWrite)
	}
}

func (sd *Sched) tickAllProcs() {
	for _, p := range append(sd.activeQ.getQ(), sd.blockedQ.getQ()...) {
		p.ticksPassed += 1
	}
}

func (sd *Sched) runLBConn() {

	// listen to messages
	for {
		msg := <-sd.lbConnRecv
		switch msg.msgType {
		case LB_M_PLACE_PROC:
			sd.activeQ.enq(msg.proc)
			msg.wg.Done()
		}
	}

}

func (sd *Sched) memUsed() Tmem {
	memUsed := Tmem(0)
	for _, p := range append(sd.activeQ.getQ(), sd.blockedQ.getQ()...) {
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
	for _, p := range append(sd.activeQ.getQ(), sd.blockedQ.getQ()...) {
		totalTicks += p.effectiveSla()
	}
	return float64(totalTicks)
}

func (sd *Sched) expectedCompInQ() float64 {
	totalTicks := Tftick(0)
	for _, p := range append(sd.activeQ.getQ(), sd.blockedQ.getQ()...) {
		totalTicks += p.profilingExpectedCompLeft()
	}
	return float64(totalTicks)
}

// expected based on profiling info
func (sd *Sched) expectedCompInActiveQ() float64 {
	totalTicks := Tftick(0)
	for _, p := range append(sd.activeQ.getQ(), sd.blockedQ.getQ()...) {
		totalTicks += p.profilingExpectedCompLeft()
	}
	return float64(totalTicks)
}

func (sd *Sched) procsInRange(sla Tftick) int {
	slaBottom := getRangeBottomFromSLA(sla)
	numProcs := 0
	for _, p := range append(sd.activeQ.getQ(), sd.blockedQ.getQ()...) {
		if getRangeBottomFromSLA(p.effectiveSla()) == slaBottom {
			numProcs += 1
		}
	}
	return numProcs
}

func (sd *Sched) maxRatioTicksPassedToSla() float64 {
	max := 0.0
	for _, p := range append(sd.activeQ.getQ(), sd.blockedQ.getQ()...) {
		if float64(p.ticksPassed/p.effectiveSla()) > max {
			max = float64(p.ticksPassed / p.effectiveSla())
		}
	}
	return max
}

func (sd *Sched) tryDeqFromBlockedQ(fTicksPassed Tftick) bool {
	// for each proc in blockedQ, see if it has become unblocked (and if yes, remove it from blockedQ and add it to activeQ)
	found := false

	for _, p := range sd.blockedQ.getQ() {
		if p.procInternals.nextUnblockedAt < (p.ticksPassed + fTicksPassed) {
			sd.blockedQ.remove(p)
			sd.activeQ.enq(p)
			found = true
		}
	}

	return found

}

// do numCores ticks of computation (only on procs in the activeQ)
func (sd *Sched) simulateRunProcs() {

	if VERBOSE_MACHINE_USAGE_STATS {
		toWrite := fmt.Sprintf("%v, %v, %.2f, %.2f, %v, %v", sd.currTick, sd.machineId,
			sd.maxRatioTicksPassedToSla(), sd.memUsage(), sd.activeQ.qlen(), sd.ticksInQ())
		logWrite(USAGE, toWrite)
	}

	ticksLeftToGive := Tftick(1)

	toWrite := fmt.Sprintf("%v, %v, curr q ACTIVE: %v, curr q BLOCKED: %v \n", sd.currTick, sd.machineId, sd.activeQ.String(), sd.blockedQ.String())
	logWrite(SCHED, toWrite)

	for ticksLeftToGive-Tftick(TICK_SCHED_THRESHOLD) > 0.0 && (sd.activeQ.qlen()+sd.blockedQ.qlen()) > 0 {

		var procToRun *Proc
		// check all blocked procs and move to active q if possible, before choosing next proc to run
		sd.tryDeqFromBlockedQ(Tftick(1) - ticksLeftToGive)
		if sd.activeQ.qlen() == 0 {
			// were not able to find anything to run - have some time pass and try again, if there are blocked procs
			toWrite := fmt.Sprintf("%v, %v, skipping time because everyone blocked, w/ ticksLeft %v \n", sd.currTick, sd.machineId, ticksLeftToGive)
			logWrite(SCHED, toWrite)
			ticksLeftToGive -= 0.1
			continue
		} else {
			procToRun = sd.activeQ.deq()
		}

		ticksUsed, done := procToRun.runTillOutOrDone(ticksLeftToGive, Tftick(1)-ticksLeftToGive)

		toWrite := fmt.Sprintf("%v, %v, running proc %v, gave %v ticks, used %v ticks\n", sd.currTick, sd.machineId, procToRun.String(), ticksLeftToGive.String(), ticksUsed.String())
		logWrite(SCHED, toWrite)

		// proc is blocked; add it to blocked Q
		if (ticksUsed < ticksLeftToGive) && !done {
			toWrite := fmt.Sprintf("noting proc is blocked, with nextUnblockedAt %v \n", procToRun.procInternals.nextUnblockedAt)
			logWrite(SCHED, toWrite)
			sd.blockedQ.enq(procToRun)
			ticksLeftToGive -= ticksUsed
			continue
		}

		ticksLeftToGive -= ticksUsed

		if !done {
			// check if the memroy used by the proc sent us over the edge (and if yes, kill as needed)
			if sd.memUsed() > MAX_MEM_PER_MACHINE {
				fmt.Printf("--> OUT OF MEMORY ON MACHINE %v\n ", sd.machineId)
				fmt.Printf("q: %v\n", sd.activeQ.String())
			}
			// add proc back into queue
			sd.activeQ.enq(procToRun)
		} else {
			// if the proc is done, update the ticksPassed to be exact for metrics etc
			procToRun.ticksPassed = procToRun.ticksPassed + (1 - ticksLeftToGive)
			// don't need to wait if we are just telling it a proc is done
			sd.lbConnSend <- &Message{sd.machineId, M_LB_PROC_DONE, procToRun, nil}
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
