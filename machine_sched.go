package slasched

import (
	"fmt"
	"math"
)

const (
	// ooh make this be based on ratio between time spent waiting and sla? (eg if I've waited for more then 1/3 of the sla, just push me)
	SLA_PUSH_THRESHOLD = 1 // 1 tick = 100 ms ==> 5 ms (see website.go)
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

func (sd *Sched) printAllProcs() {
	for _, p := range sd.q.getQ() {
		toWrite := fmt.Sprintf("%v, %v, -1, %v, %v, %v\n", sd.currTick, sd.machineId,
			float64(p.procInternals.sla), float64(p.procInternals.actualComp), float64(p.compUsed()))
		logWrite(CURR_PROCS, toWrite)
	}
	for _, core := range sd.coreScheds {
		core.printAllProcs()
	}
}

func (sd *Sched) tickAllProcs() {
	for _, p := range sd.q.getQ() {
		p.ticksPassed += 1
	}
	for _, core := range sd.coreScheds {
		core.tickAllProcs()
	}
}

func (sd *Sched) runLBConn() {

	// listen to messages
	for {
		msg := <-sd.lbConnRecv
		switch msg.msgType {
		case LB_M_PLACE_PROC:
			if msg.proc.effectiveSla() < SLA_PUSH_THRESHOLD {
				// place on core with min ticksInQ
				minVal := int(math.Inf(1))
				var coreToUse *CoreSched
				for _, c := range sd.coreScheds {
					// core is a contender if has memory for it
					if (MAX_MEM_PER_CORE - c.memUsed()) > Tmem(msg.proc.procTypeProfile.memUsg.avg+msg.proc.procTypeProfile.memUsg.stdDev) {
						// want core with min procs in range or empty Q
						if (c.procsInRange(msg.proc.effectiveSla()) < minVal) ||
							(c.procsInRange(msg.proc.effectiveSla()) == minVal && c.ticksInQ() == 0) {
							minVal = c.procsInRange(msg.proc.effectiveSla())
							coreToUse = c
						}
					}
				}
				// is this safe? -> yes, because I only add in between ticks, not while they're running
				if coreToUse != nil {
					toWrite := fmt.Sprintf("%v, %v, %v, machine pushing proc to core: %v \n", sd.currTick, sd.machineId, coreToUse.coreId, msg.proc.String())
					logWrite(SCHED, toWrite)
					sd.coreScheds[coreToUse.coreId].q.enq(msg.proc)
					toWrite = fmt.Sprintf("new Q:  %v\n", sd.coreScheds[coreToUse.coreId].q.String())
					logWrite(SCHED, toWrite)
				} else {
					sd.q.enq(msg.proc)
				}
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
			memFree := MAX_MEM_PER_CORE - sd.coreScheds[msg.sender].memUsed()
			// if global q has work that fits, steal that
			if p := sd.q.deq(); p != nil {
				if (p.procTypeProfile.memUsg.avg + p.procTypeProfile.memUsg.stdDev) < float64(memFree) {
					toWrite := fmt.Sprintf("%v, %v, %v, machine giving proc from own q: %v \n", sd.currTick, sd.machineId, msg.sender, p.String())
					logWrite(SCHED, toWrite)
					sd.coreConnSend[msg.sender] <- &Message{sd.machineId, M_C_PUSH_PROC, p, nil}
					continue
				} else {
					sd.q.enq(p)
				}
			}

			// otherwise look for another core that might have work
			var procToSteal *Proc
			var coreWithMaxWork *CoreSched
			maxTicks := Tftick(0)
			for _, c := range sd.coreScheds {
				if c.ticksInQ() > maxTicks {
					maxTicks = c.ticksInQ()
					coreWithMaxWork = c
				}
			}
			if coreWithMaxWork != nil {
				procToSteal = coreWithMaxWork.q.workSteal(memFree)
			}
			if procToSteal != nil {
				toWrite := fmt.Sprintf("%v, %v, %v, machine giving proc from other cores q: %v \n", sd.currTick, sd.machineId, msg.sender, procToSteal.String())
				logWrite(SCHED, toWrite)
			}
			sd.coreConnSend[msg.sender] <- &Message{sd.machineId, M_C_PUSH_PROC, procToSteal, nil}
		case C_M_PROC_DONE:
			sd.lbConnSend <- &Message{sd.machineId, M_LB_PROC_DONE, msg.proc, nil}
		}
	}
}

func (sd *Sched) tick() {

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

func (sd *Sched) procsInRange(sla Tftick) int {
	slaBottom := getRangeBottomFromSLA(sla)
	numProcs := 0
	for _, p := range sd.q.getQ() {
		if getRangeBottomFromSLA(p.effectiveSla()) == slaBottom {
			numProcs += 1
		}
	}
	for _, core := range sd.coreScheds {
		numProcs += core.procsInRange(sla)
	}
	return numProcs
}
