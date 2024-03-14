package slasched

import "fmt"

const (
	EWMA_ALPHA = 0.2
)

type Sched struct {
	machineId  Tid
	numCores   int
	coreScheds map[Tid]*CoreSched
	q          *Queue
	lbConn     chan *MachineMessages
	currTick   int
}

func newSched(lbConn chan *MachineMessages, mid Tid, numCores int) *Sched {
	sd := &Sched{
		machineId: mid,
		numCores:  numCores,
		q:         newQueue(),
		lbConn:    lbConn,
		currTick:  0,
	}
	coreScheds := map[Tid]*CoreSched{}
	for i := 0; i < numCores; i++ {
		coreChan := make(chan *CoreMessages)
		coreScheds[Tid(i)] = newCoreSched(coreChan, mid, Tid(i))
		go sd.runCoreConn(coreChan)
	}
	sd.coreScheds = coreScheds
	return sd
}

func (sd *Sched) String() string {
	return fmt.Sprintf("machine scheduler: %v", sd.machineId)
}

func (sd *Sched) runCoreConn(coreChan chan *CoreMessages) {

	for {
		msg := <-coreChan
		switch msg.msgType {
		case NEED_WORK:
			coreChan <- &CoreMessages{PUSH_PROC, sd.q.deq()}
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
	for _, core := range sd.coreScheds {
		totalTicks += core.ticksInQ()
	}
	return float64(totalTicks)
}
