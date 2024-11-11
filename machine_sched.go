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
	numCores   int
	activeQ    *Queue
	lbConnSend chan *Message // channel to send messages to LB
	lbConnRecv chan *Message // channel this machine recevies messages on from the LB
	currTick   Tftick
}

func newSched(numCores int, lbConnSend chan *Message, lbConnRecv chan *Message, mid Tid) *Sched {
	sd := &Sched{
		machineId:  mid,
		numCores:   numCores,
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
	sd.simulateRunProcs()
	sd.currTick += 1
}

func (sd *Sched) printAllProcs() {

	for _, p := range sd.activeQ.getQ() {
		toWrite := fmt.Sprintf("%v, %v, 1, %v, %v, %v\n", int(sd.currTick), sd.machineId,
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

	coreToRunningWaitTime := make(map[int]Tftick)
	for i := 0; i < sd.numCores; i++ {
		coreToRunningWaitTime[i] = Tftick(0)
	}

	getAddMinRunningWaitTime := func(toAdd Tftick) Tftick {
		minVal := Tftick(math.MaxFloat64)
		minCore := -1
		for i := 0; i < sd.numCores; i++ {
			if coreToRunningWaitTime[i] < minVal {
				minVal = coreToRunningWaitTime[i]
				minCore = i
			}
		}
		// ofstream sched_file;
		// sched_file.open("../sched.txt", std::ios_base::app);
		// sched_file << "adding ceil " << to_add << " to core " << min_core << ", whose waiting time is thus now " << cores_to_running_waiting_time.at(min_core) + to_add << endl;
		// sched_file.close();
		coreToRunningWaitTime[minCore] += toAdd
		return minVal
	}

	for _, p := range fullList {

		waitTime := getAddMinRunningWaitTime(p.getExpectedCompLeft())
		if p.getSlack(sd.currTick)-waitTime < 0.0 {
			return false
		}
	}

	return true

}

// do numCores ticks of computation (only on procs in the activeQ)
func (sd *Sched) simulateRunProcs() {

	if VERBOSE_MACHINE_USAGE_STATS {
		toWrite := fmt.Sprintf("%v, %v,  %.2f, %v", int(sd.currTick), sd.machineId, sd.memUsage(), sd.activeQ.qlen())
		logWrite(USAGE, toWrite)
	}

	totalTicksLeftToGive := Tftick(sd.numCores)
	ticksLeftPerCore := make(map[int]Tftick, 0)
	for i := 0; i < sd.numCores; i++ {
		ticksLeftPerCore[i] = Tftick(1)
	}

	toWrite := fmt.Sprintf("%v, %v, curr q ACTIVE: %v \n", int(sd.currTick), sd.machineId, sd.activeQ.String())
	logWrite(SCHED, toWrite)

	var toReq []*Proc
	// TODO: switch the loop order? oh but then maybe a proc will be processed later that actually would have been in parallel
	// I should work through some examples
	for totalTicksLeftToGive-Tftick(TICK_SCHED_THRESHOLD) > 0.0 && sd.activeQ.qlen() > 0 {

		for currCore := 0; currCore < sd.numCores; currCore++ {
			procToRun := sd.activeQ.deq()

			if procToRun == nil {
				break
			}

			ticksUsed, done := procToRun.runTillOutOrDone(ticksLeftPerCore[currCore])

			toWrite := fmt.Sprintf("%v, %v, %v, running proc %v, gave %v ticks, used %v ticks\n", sd.currTick, sd.machineId, currCore, procToRun.String(), ticksLeftPerCore[currCore].String(), ticksUsed.String())
			logWrite(SCHED, toWrite)

			ticksLeftPerCore[currCore] -= ticksUsed
			totalTicksLeftToGive -= ticksUsed

			if !done {
				// check if the memroy used by the proc sent us over the edge (and if yes, kill as needed)
				if sd.memUsed() > MAX_MEM_PER_MACHINE {
					fmt.Printf("--> OUT OF MEMORY ON MACHINE %v\n ", sd.machineId)
					fmt.Printf("q: %v\n", sd.activeQ.String())
				}
				// add proc back into queue (a temporary one, b/c once a proc has run on one core it shouldn't be allowed to get time on another core also,
				// and since there is no blocking/waking during a tick we know it wouldn't magically become the one to run again later so can just wait till the end of the tick)
				toReq = append(toReq, procToRun)
			} else {
				// if the proc is done, update the ticksPassed to be exact for metrics etc
				procToRun.timeDone = sd.currTick + (1 - ticksLeftPerCore[currCore])
				// don't need to wait if we are just telling it a proc is done
				sd.lbConnSend <- &Message{sd.machineId, M_LB_PROC_DONE, procToRun, nil}
			}

		}

	}

	// procs that have been run can now be re-added to the q
	for _, p := range toReq {
		sd.activeQ.enq(p)
	}

	// this is dumb but make accounting for util easier
	if totalTicksLeftToGive < 0.00002 {
		totalTicksLeftToGive = 0
	}
	if VERBOSE_MACHINE_USAGE_STATS {
		toWrite := fmt.Sprintf(", %v\n", float64(math.Copysign(float64(totalTicksLeftToGive), 1)))
		logWrite(USAGE, toWrite)
	}
}
