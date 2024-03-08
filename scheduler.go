package slasched

import "fmt"

const (
	EWMA_ALPHA = 0.2
)

type CoreMsgType int // messages between the dispatcher and core scheduler

const (
	NEED_WORK      CoreMsgType = iota // msg by core bc it is out of work
	PUSH_PROC                         // msg by the controller to the core with a proc it should take
	RETURNING_PROC                    // potentially if proc pushed to core it responds with a proc it is then returning
)

type CoreMessages struct {
	msgType CoreMsgType
	proc    *Proc
}

type Sched struct {
	machineId  Tid
	numCores   int
	coreScheds []*CoreSched
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
	coreScheds := []*CoreSched{}
	for i := 0; i < numCores; i++ {
		coreChan := make(chan *CoreMessages)
		coreScheds = append(coreScheds, newCoreSched(coreChan, mid, Tid(i)))
		go sd.runDispatcher(lbConn, coreChan)
	}
	sd.coreScheds = coreScheds
	return sd
}

func (sd *Sched) String() string {
	return fmt.Sprintf("machine scheduler: %v", sd.machineId)
}

func (sd *Sched) printAllProcs() {
	// print all in global q
	for _, p := range sd.q.q {
		fmt.Printf("current: %v, %v, %v, %v, %v\n", sd.currTick, sd.machineId, float64(p.procInternals.sla), float64(p.procInternals.actualComp), float64(p.procInternals.compDone))
	}

	// then all in the local core qs
	for _, cs := range sd.coreScheds {
		cs.printAllProcs()
	}
}

func (sd *Sched) runDispatcher(lbConn chan *MachineMessages, coreChan chan *CoreMessages) {

	for {
		select {
		case msg := <-coreChan:
			switch msg.msgType {
			case NEED_WORK:
				fmt.Println("giving core work")
				coreChan <- &CoreMessages{PUSH_PROC, sd.q.deq()}
			}
		case msg := <-lbConn:
			switch msg.msgType {
			case ADD_PROC:
				fmt.Println("got proc from lb")
				sd.q.enq(msg.proc)
			}
		}
	}

}

func (sd *Sched) tick() {
	sd.currTick += 1
	for _, cs := range sd.coreScheds {
		cs.tick()
	}
}
