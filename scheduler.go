package slasched

import (
	"fmt"
	"math"

	"golang.org/x/exp/maps"
)

const (
	EWMA_ALPHA             = 0.2
	FRACTION_SHINJUKU      = 0.3
	SLA_THRESHOLD_SHINJUKU = 1
	MAX_PROCS_PER_PS_CORE  = 5
)

type CoreMsgType int // messages between the dispatcher and core scheduler

const (
	NEED_WORK      CoreMsgType = iota // msg by core bc it is out of work
	PUSH_PROC                         // msg by the controller to the core with a proc it should take
	RETURNING_PROC                    // potentially if proc pushed to core it responds with a proc it is then returning
)

type CoreMessages struct {
	msgType   CoreMsgType
	proc      *Proc
	schedType SchedulerType
}

type Sched struct {
	machineId          Tid
	numCores           int
	coreSchedsPS       map[int]*CoreSched
	coreSchedsShinjuku map[int]*CoreSched
	q_ps               *Queue
	q_shinjuku         *Queue
	lbConn             chan *MachineMessages
	currTick           int
}

func newSched(lbConn chan *MachineMessages, mid Tid, numCores int) *Sched {
	sd := &Sched{
		machineId:  mid,
		numCores:   numCores,
		q_ps:       newQueue(),
		q_shinjuku: newQueue(),
		lbConn:     lbConn,
		currTick:   0,
	}
	sd.coreSchedsPS = make(map[int]*CoreSched, 0)
	sd.coreSchedsShinjuku = make(map[int]*CoreSched, 0)
	for i := 0; i < numCores; i++ {
		coreChan := make(chan *CoreMessages)
		if float64(i) < float64(numCores)*FRACTION_SHINJUKU {
			sd.coreSchedsShinjuku[i] = newCoreSched(coreChan, mid, Tid(i), SHINJUKU)
		} else {
			sd.coreSchedsPS[i] = newCoreSched(coreChan, mid, Tid(i), PS)
		}
		go sd.runDispatcher(lbConn, coreChan)
	}
	return sd
}

func (sd *Sched) String() string {
	return fmt.Sprintf("machine scheduler: %v", sd.machineId)
}

func (sd *Sched) printAllProcs() {
	// print all in global q
	for _, p := range append(sd.q_ps.q, sd.q_shinjuku.q...) {
		fmt.Printf("current: %v, %v, -1, %v, %v, %v\n", sd.currTick, sd.machineId, float64(p.procInternals.sla), float64(p.procInternals.actualComp), float64(p.procInternals.compDone))
	}

	// then all in the local core qs
	for _, cs := range append(maps.Values(sd.coreSchedsShinjuku), maps.Values(sd.coreSchedsPS)...) {
		cs.printAllProcs()
	}
}

func (sd *Sched) runDispatcher(lbConn chan *MachineMessages, coreChan chan *CoreMessages) {

	for {
		select {
		case msg := <-coreChan:
			switch msg.msgType {
			case NEED_WORK:
				switch msg.schedType {
				case PS:
					if sd.q_ps.qlen() < 0 {
						coreChan <- &CoreMessages{PUSH_PROC, sd.q_ps.deq(), PS}
					} else {
						if ind := sd.getPSCoreWithMaxWork(); ind > 0 {
							procToSteal := sd.coreSchedsPS[ind].coreQ.deq()
							coreChan <- &CoreMessages{PUSH_PROC, procToSteal, PS}
						} else {
							coreChan <- &CoreMessages{PUSH_PROC, nil, PS}
						}
					}

				case SHINJUKU:
					coreChan <- &CoreMessages{PUSH_PROC, sd.q_shinjuku.deq(), SHINJUKU}
				}
			}
		case msg := <-lbConn:
			switch msg.msgType {
			case ADD_PROC:
				fmt.Printf("adding: %v, %v, %v, %v\n", sd.currTick, sd.machineId, float64(msg.proc.procInternals.sla), float64(msg.proc.procInternals.actualComp))
				if msg.proc.getSla() < SLA_THRESHOLD_SHINJUKU {
					sd.q_shinjuku.enq(msg.proc)
				} else {
					// just put on a core
					coreInd := sd.getPSCoreWithMinWork()
					sd.coreSchedsPS[coreInd].coreQ.enq(msg.proc)
				}
				msg.wg.Done()
			}
		}
	}

}

func (sd *Sched) getPSCoreWithMinWork() int {
	minNumProcs := int(math.Inf(1))
	coreInd := -1
	for i, core := range sd.coreSchedsPS {
		if core.coreQ.qlen() < minNumProcs {
			minNumProcs = core.coreQ.qlen()
			coreInd = i
		}
	}
	return coreInd
}

func (sd *Sched) getPSCoreWithMaxWork() int {
	maxNumProcs := 0
	coreInd := -1
	for i, core := range sd.coreSchedsPS {
		if core.coreQ.qlen() > maxNumProcs {
			maxNumProcs = core.coreQ.qlen()
			coreInd = i
		}
	}
	return coreInd
}

func (sd *Sched) tick() {
	for _, cs := range append(maps.Values(sd.coreSchedsShinjuku), maps.Values(sd.coreSchedsPS)...) {
		cs.tick()
	}
	sd.currTick += 1
}
