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
	activeQ    map[Tid]*Queue
	idleHeap   *IdleHeap
	lbConnSend chan *Message // channel to send messages to LB
	lbConnRecv chan *Message // channel this machine recevies messages on from the LB
	currTick   Tftick
}

func newSched(numCores int, idleHeap *IdleHeap, lbConnSend chan *Message, lbConnRecv chan *Message, mid Tid) *Sched {
	sd := &Sched{
		machineId:  mid,
		numCores:   numCores,
		activeQ:    make(map[Tid]*Queue),
		idleHeap:   idleHeap,
		lbConnSend: lbConnSend,
		lbConnRecv: lbConnRecv,
		currTick:   0,
	}
	for i := 0; i < numCores; i++ {
		sd.activeQ[Tid(i)] = newQueue()
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

	for i := 0; i < sd.numCores; i++ {
		for _, p := range sd.activeQ[Tid(i)].getQ() {
			toWrite := fmt.Sprintf("%v, %v, 1, %v, %v, %v\n", int(sd.currTick), sd.machineId,
				float64(p.deadline), float64(p.procInternals.actualComp), float64(p.compUsed()))
			logWrite(CURR_PROCS, toWrite)
		}
	}
}

func (sd *Sched) runLBConn() {

	// listen to messages
	for {
		msg := <-sd.lbConnRecv
		switch msg.msgType {
		case LB_M_PLACE_PROC:
			sd.pickCorePlaceProc(msg.proc)
			msg.wg.Done()
		}
	}

}

func (sd *Sched) pickCorePlaceProc(newProc *Proc) {

	// metrics by which to pick the most suitable core:
	// - most amount of hol slack?

	possCores := make(map[Tid]Tftick)

	for currCore := 0; currCore < sd.numCores; currCore++ {

		fullList := append(sd.activeQ[Tid(currCore)].getQ(), newProc)
		sort.Slice(fullList, func(i, j int) bool {
			return fullList[i].deadline < fullList[j].deadline
		})

		runningWaitTime := Tftick(0)
		extraSlack := Tftick(1)

		for _, p := range fullList {

			runningWaitTime += p.getExpectedCompLeft()
			if p.getSlack(sd.currTick)-runningWaitTime < 0.0 {
				continue
			}
			if p.getSlack(sd.currTick)-runningWaitTime < extraSlack {
				extraSlack = p.getSlack(sd.currTick) - runningWaitTime
			}
		}

		possCores[Tid(currCore)] = extraSlack

	}

	minVal := Tftick(math.MaxFloat64)
	var minCore Tid
	for pc, extra := range possCores {
		if extra < minVal {
			minCore = pc
			minVal = extra
		}
	}

	sd.activeQ[minCore].enq(newProc)

}

// checks if a proc can fit:
// a) if it has enough slack to accomodate procs with a lower deadline, and
// b) if procs with a larger deadline have enough slack to accomodate it
func (sd *Sched) okToPlace(newProc *Proc) bool {

	// fmt.Printf("--- running okToPlace: %v, %v \n", sd.currTick, sd.machineId)

	for currCore := 0; currCore < sd.numCores; currCore++ {

		fullList := append(sd.activeQ[Tid(currCore)].getQ(), newProc)
		sort.Slice(fullList, func(i, j int) bool {
			return fullList[i].deadline < fullList[j].deadline
		})

		runningWaitTime := Tftick(0)

		for _, p := range fullList {

			runningWaitTime += p.getExpectedCompLeft()
			if p.getSlack(sd.currTick)-runningWaitTime < 0.0 {
				continue
			}
		}

		return true

	}

	return false

}

// do numCores ticks of computation (only on procs in the activeQ)
func (sd *Sched) simulateRunProcs() {

	sum_qlens := 0
	for i := 0; i < sd.numCores; i++ {
		sum_qlens += sd.activeQ[Tid(i)].qlen()
	}

	if VERBOSE_MACHINE_USAGE_STATS {
		toWrite := fmt.Sprintf("%v, %v, %v", int(sd.currTick), sd.machineId, sum_qlens)
		logWrite(USAGE, toWrite)
	}

	ticksLeftPerCore := make(map[int]Tftick, 0)
	totalTicksLeftToGive := Tftick(sd.numCores)

	for currCore := 0; currCore < sd.numCores; currCore++ {

		ticksLeftToGive := Tftick(1)
		ticksLeftPerCore[currCore] = Tftick(1)

		toWrite := fmt.Sprintf("%v, %v, %v, curr q ACTIVE: %v \n", int(sd.currTick), sd.machineId, currCore, sd.activeQ[Tid(currCore)].String())
		logWrite(SCHED, toWrite)

		for ticksLeftToGive-Tftick(TICK_SCHED_THRESHOLD) > 0.0 && sd.activeQ[Tid(currCore)].qlen() > 0 {

			procToRun := sd.activeQ[Tid(currCore)].deq()

			if procToRun == nil {
				break
			}

			ticksUsed, done := procToRun.runTillOutOrDone(ticksLeftPerCore[currCore])

			toWrite := fmt.Sprintf("%v, %v, %v, running proc %v, gave %v ticks, used %v ticks\n", int(sd.currTick), sd.machineId, currCore, procToRun.String(), ticksLeftPerCore[currCore].String(), ticksUsed.String())
			logWrite(SCHED, toWrite)

			ticksLeftPerCore[currCore] -= ticksUsed
			ticksLeftToGive -= ticksUsed
			totalTicksLeftToGive -= ticksUsed

			if !done {
				sd.activeQ[Tid(currCore)].enq(procToRun)
			} else {
				// if the proc is done, update the ticksPassed to be exact for metrics etc
				procToRun.timeDone = sd.currTick + (1 - ticksLeftPerCore[currCore])
				// don't need to wait if we are just telling it a proc is done
				sd.lbConnSend <- &Message{sd.machineId, M_LB_PROC_DONE, procToRun, nil}
			}
		}
	}

	// do this for every core
	for coreNum := 0; coreNum < sd.numCores; coreNum++ {
		// use core num to get info
		if sd.activeQ[Tid(coreNum)].getHOLSlack(sd.currTick+1) > IDLE_HEAP_THRESHOLD {

			sd.idleHeap.lock.Lock()
			// also if it is already in the heap, then replace it with the new value
			if contains(sd.idleHeap.heap, TmachineCoreId{sd.machineId, coreNum}) {
				remove(sd.idleHeap.heap, TmachineCoreId{sd.machineId, coreNum})
			}
			toPush := TIdleMachine{
				compIdleFor:   sd.activeQ[Tid(coreNum)].getHOLSlack(sd.currTick + 1),
				machineCoreId: TmachineCoreId{sd.machineId, coreNum},
			}
			sd.idleHeap.heap.Push(toPush)
			sd.idleHeap.lock.Unlock()
		}
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
