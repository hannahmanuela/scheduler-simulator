package slasched

import (
	"container/heap"
	"fmt"
	"sync"
)

type MsgType int

const (
	M_LB_PROC_DONE  MsgType = iota // machine -> lb; proc is done
	LB_M_PLACE_PROC                // lb -> machine; placing proc on machine
	C_M_NEED_WORK                  // core -> machine; core is out of work
	C_M_PROC_DONE                  // core -> machine; proc is done
	M_C_PUSH_PROC                  // machine -> core; proc for core
)

type Message struct {
	sender  Tid
	msgType MsgType
	proc    *Proc
	wg      *sync.WaitGroup
}

type TIdleMachine struct {
	compIdleFor Tftick // The key used for sorting.
	machineId   Tid    // The value associated with the key.
}
type MinHeap []TIdleMachine

func (h MinHeap) Len() int           { return len(h) }
func (h MinHeap) Less(i, j int) bool { return h[i].compIdleFor < h[j].compIdleFor }
func (h MinHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *MinHeap) Push(x any)        { *h = append(*h, x.(TIdleMachine)) }

func (h *MinHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func popNextLarger(h *MinHeap, value Tftick) (TIdleMachine, bool) {
	var tempHeap MinHeap

	for h.Len() > 0 {
		item := heap.Pop(h).(TIdleMachine)

		if item.compIdleFor > value {
			return item, true
		}
		tempHeap = append(tempHeap, item)
	}

	for _, item := range tempHeap {
		heap.Push(h, item)
	}
	return TIdleMachine{}, false
}

type IdleHeap struct {
	heap *MinHeap
	lock sync.RWMutex
}

type GlobalSched struct {
	machines        map[Tid]*Machine
	idleMachines    *IdleHeap
	procq           *Queue
	machineConnRecv chan *Message         // listen for messages from machines
	machineConnSend map[Tid]chan *Message // send messages to machine
	currTick        int
}

func newLoadBalancer(machines map[Tid]*Machine, idleHeap *IdleHeap, lbSendToMachines map[Tid]chan *Message, lbRecv chan *Message) *GlobalSched {
	lb := &GlobalSched{
		machines:        machines,
		idleMachines:    idleHeap,
		procq:           &Queue{q: make([]*Proc, 0)},
		machineConnRecv: lbRecv,
		machineConnSend: lbSendToMachines,
		currTick:        0,
	}

	go lb.listenForMachineMessages()
	return lb
}

func (gs *GlobalSched) MachinesString() string {
	str := "machines: \n"
	for _, m := range gs.machines {
		str += "   " + m.String()
	}
	return str
}

func (lb *GlobalSched) listenForMachineMessages() {
	for {
		msg := <-lb.machineConnRecv
		switch msg.msgType {
		case M_LB_PROC_DONE:
			if VERBOSE_LB_STATS {
				toWrite := fmt.Sprintf("%v, %v, %v, %v, %v, %v\n", lb.currTick, msg.proc.machineId, msg.proc.procInternals.procType, float64(msg.proc.deadline), float64(msg.proc.timeDone-msg.proc.timeStarted), float64(msg.proc.procInternals.actualComp))
				logWrite(DONE_PROCS, toWrite)
			}
		}
	}
}

func (lb *GlobalSched) placeProcs() {
	// setup
	lb.currTick += 1
	p := lb.getProc()

	toReQ := make([]*Proc, 0)

	for p != nil {
		// place given proc

		machineToUse := lb.pickMachine(p)

		if machineToUse == nil {
			toReQ = append(toReQ, p)
			p = lb.getProc()
			continue
		}

		// place proc on chosen machine
		p.machineId = machineToUse.mid
		var wg sync.WaitGroup
		wg.Add(1)
		lb.machineConnSend[machineToUse.mid] <- &Message{-1, LB_M_PLACE_PROC, p, &wg}
		wg.Wait()
		if VERBOSE_LB_STATS {
			toWrite := fmt.Sprintf("%v, %v, %v, %v, %v\n", lb.currTick, machineToUse.mid, p.procInternals.procType, float64(p.procInternals.deadline), float64(p.procInternals.actualComp))
			logWrite(ADDED_PROCS, toWrite)
		}
		p = lb.getProc()
	}

	for _, p := range toReQ {
		lb.putProc(p)
	}
}

// admission control:
// 1. first look for machines that simply currently have the space (using interval tree of immediately available compute)
// 2. if there are none, do the ok to place call on all machines? on some machines? just random would be the closest to strictly following what they do...
func (lb *GlobalSched) pickMachine(procToPlace *Proc) *Machine {

	lb.idleMachines.lock.Lock()
	machine, found := popNextLarger(lb.idleMachines.heap, procToPlace.maxComp)
	lb.idleMachines.lock.Unlock()
	if found {
		toWrite := fmt.Sprintf("%v, LB placing proc: %v, found an idle machine %d with %.2f comp avail \n", lb.currTick, procToPlace.String(), machine.machineId, float64(machine.compIdleFor))
		logWrite(SCHED, toWrite)

		return lb.machines[machine.machineId]
	}

	// if no idle machine, use power of k choices (for now k = number of machines :D )
	var machineToUse *Machine
	contenderMachines := make([]*Machine, 0)

	for _, m := range lb.machines {
		if m.sched.okToPlace(procToPlace) {
			contenderMachines = append(contenderMachines, m)
		}
	}

	toWrite := fmt.Sprintf("%v, LB placing proc: %v, there are %v contender machines \n", lb.currTick, procToPlace.String(), len(contenderMachines))
	logWrite(SCHED, toWrite)

	if len(contenderMachines) == 0 {
		fmt.Printf("%v: DOESN'T FIT ANYWHERE :(( -- re-enqing: %v \n", lb.currTick, procToPlace)
		return nil
	}

	// TODO: this is stupid
	machineToUse = contenderMachines[0]

	return machineToUse
}

func (lb *GlobalSched) getProc() *Proc {
	return lb.procq.deq()
}

func (lb *GlobalSched) putProc(proc *Proc) {
	lb.procq.enq(proc)
}
