package slasched

import (
	"fmt"
	"math"
	"sort"
)

const (
	TICK_SCHED_THRESHOLD = 0.00001 // amount of ticks after which I stop scheduling; given that 1 tick = 100ms (see website.go)
)

type Sched struct {
	machineId  Tid
	activeQ    *Queue
	lbConnSend chan *Message // channel to send messages to LB
	lbConnRecv chan *Message // channel this machine recevies messages on from the LB
	currTick   Tftick
}

func newSched(lbConnSend chan *Message, lbConnRecv chan *Message, mid Tid) *Sched {
	sd := &Sched{
		machineId:  mid,
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

	sd.simulateRunProcs()
}

func (sd *Sched) printAllProcs() {

	for _, p := range sd.activeQ.getQ() {
		toWrite := fmt.Sprintf("%v, %v, 1, %v, %v, %v\n", sd.currTick, sd.machineId,
			float64(p.deadline), float64(p.procInternals.actualComp), float64(p.compUsed()))
		logWrite(CURR_PROCS, toWrite)
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

// checks if a proc can fit:
// a) if it has enough slack to accomodate procs with a lower deadline, and
// b) if procs with a larger deadline have enough slack to accomodate it
func (sd *Sched) okToPlace(newProc *Proc) bool {

	// fmt.Printf("--- running okToPlace: %v, %v \n", sd.currTick, sd.machineId)

	fullList := append(sd.activeQ.getQ(), newProc)
	sort.Slice(fullList, func(i, j int) bool {
		return fullList[i].deadline < fullList[j].deadline
	})

	runningWaitTime := Tftick(0)

	for _, p := range fullList {
		// make sure that the current proc is able to wait for all the prvious procs
		pSlack := p.getSlack(sd.currTick)

		// fmt.Printf("%v, %v: running wait time: %v curr proc deadline: %v, curr proc sla: %v, expectedCompLeft: %v, slack: %v\n",
		// 	sd.currTick, sd.machineId, runningWaitTime, p.deadline, p.getSla(), p.expectedTimeLeft(), pSlack)

		if pSlack-runningWaitTime < 0.0 {
			// fmt.Println("returning false")
			return false
		}
		// add current proc to runningWaitTime
		runningWaitTime += p.getExpectedCompLeft()
	}

	// fmt.Println("returning true")
	return true

}

// do numCores ticks of computation (only on procs in the activeQ)
func (sd *Sched) simulateRunProcs() {

	if VERBOSE_MACHINE_USAGE_STATS {
		toWrite := fmt.Sprintf("%v, %v,  %.2f, %v", sd.currTick, sd.machineId, sd.memUsage(), sd.activeQ.qlen())
		logWrite(USAGE, toWrite)
	}

	ticksLeftToGive := Tftick(1)
	unusedTicks := Tftick(0)

	toWrite := fmt.Sprintf("%v, %v, curr q ACTIVE: %v \n", sd.currTick, sd.machineId, sd.activeQ.String())
	logWrite(SCHED, toWrite)

	for ticksLeftToGive-Tftick(TICK_SCHED_THRESHOLD) > 0.0 && sd.activeQ.qlen() > 0 {

		var procToRun *Proc
		procToRun = sd.activeQ.deq()

		ticksUsed, done := procToRun.runTillOutOrDone(ticksLeftToGive)

		toWrite := fmt.Sprintf("%v, %v, running proc %v, gave %v ticks, used %v ticks\n", sd.currTick, sd.machineId, procToRun.String(), ticksLeftToGive.String(), ticksUsed.String())
		logWrite(SCHED, toWrite)

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
			procToRun.timeDone = (sd.currTick - procToRun.timeStarted) + (1 - ticksLeftToGive)
			// don't need to wait if we are just telling it a proc is done
			sd.lbConnSend <- &Message{sd.machineId, M_LB_PROC_DONE, procToRun, nil}
		}

	}

	// this is dumb but make accounting for util easier
	if ticksLeftToGive < 0.00002 {
		ticksLeftToGive = 0
	}
	if VERBOSE_MACHINE_USAGE_STATS {
		toWrite := fmt.Sprintf(", %v\n", float64(math.Copysign(float64(ticksLeftToGive+unusedTicks), 1)))
		logWrite(USAGE, toWrite)
	}
}
