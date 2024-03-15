package slasched

import (
	"fmt"
	"math"
)

const (
	SLA_PUSH_THRESHOLD = 0.01 // 1 tick = 100 ms ==> 1 ms (see website.go)
)

type Sched struct {
	machineId    Tid
	numCores     int
	coreScheds   map[Tid]*CoreSched
	coreConnRecv chan *Message
	coreConnSend map[Tid]chan *Message
	q            *Queue
	lbConnSend   chan *Message // channel to send messages to LB
	lbConnRecv   chan *Message // channel this machine recevies messages on from the LB
	currTick     int
}

func newSched(lbConnSend chan *Message, lbConnRecv chan *Message, mid Tid, numCores int) *Sched {
	sd := &Sched{
		machineId:    mid,
		numCores:     numCores,
		q:            newQueue(),
		lbConnSend:   lbConnSend,
		lbConnRecv:   lbConnRecv,
		coreConnSend: map[Tid]chan *Message{},
		currTick:     0,
	}
	coreChanRecv := make(chan *Message)
	sd.coreConnRecv = coreChanRecv
	coreScheds := map[Tid]*CoreSched{}
	for i := 0; i < numCores; i++ {
		coreId := Tid(i)
		coreChanSend := make(chan *Message)
		coreScheds[coreId] = newCoreSched(coreChanRecv, coreChanSend, mid, coreId)
		sd.coreConnSend[coreId] = coreChanSend
	}
	sd.coreScheds = coreScheds
	go sd.runLBConn()
	go sd.runCoreConn()
	return sd
}

func (sd *Sched) String() string {
	return fmt.Sprintf("machine scheduler: %v", sd.machineId)
}

func (sd *Sched) runLBConn() {

	// listen to messages
	for {
		msg := <-sd.lbConnRecv
		switch msg.msgType {
		case LB_M_PLACE_PROC:
			if msg.proc.effectiveSla() < SLA_PUSH_THRESHOLD {
				// place on core with min ticksInQ
				minVal := Tftick(math.Inf(1))
				var coreToUse *CoreSched
				for _, c := range sd.coreScheds {
					if c.ticksInQ() < minVal {
						minVal = c.ticksInQ()
						coreToUse = c
					}
				}
				// is this safe? -> yes, because I only add in between ticks, not while they're running
				sd.coreScheds[coreToUse.coreId].q.enq(msg.proc)
			} else {
				sd.q.enq(msg.proc)
			}
			msg.wg.Done()
		}
	}

}

func (sd *Sched) runCoreConn() {

	for {
		msg := <-sd.coreConnRecv
		switch msg.msgType {
		case C_M_NEED_WORK:
			// TODO: this could do cross core work stealing if there isn't any left on machine q
			sd.coreConnSend[msg.sender] <- &Message{sd.machineId, M_C_PUSH_PROC, sd.q.deq(), nil}
		case C_M_PROC_DONE:
			sd.lbConnSend <- &Message{sd.machineId, M_LB_PROC_DONE, msg.proc, nil}
		}
	}

}

func (sd *Sched) tick() {

	// place procs? or based on work stealing?
	// combo: if one of the new procs has a deadline less than the max deadline on a cpu, push it there
	// if not, just put them in q and it'll be based on work stealing

	sd.currTick += 1
	for _, cs := range sd.coreScheds {
		cs.tick()
	}
}

func (sd *Sched) memFree() float64 {
	memFree := Tmem(0)
	for _, core := range sd.coreScheds {
		memFree += MAX_MEM_PER_CORE - core.memUsed()
	}
	return float64(memFree)
}

func (sd *Sched) ticksInQ() float64 {
	totalTicks := Tftick(0)
	for _, p := range sd.q.getQ() {
		totalTicks += p.expectedCompLeft()
	}
	for _, core := range sd.coreScheds {
		totalTicks += core.ticksInQ()
	}
	return float64(totalTicks)
}
