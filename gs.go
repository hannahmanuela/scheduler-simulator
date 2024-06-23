package slasched

import (
	"fmt"
	"math"
	"os"
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

type GlobalSched struct {
	machines         map[Tid]*Machine
	procq            *Queue
	machineConnRecv  chan *Message         // listen for messages from machines
	machineConnSend  map[Tid]chan *Message // send messages to machine
	currTick         int
	procTypeProfiles map[ProcType]*ProvProcDistribution
}

func newLoadBalancer(machines map[Tid]*Machine, lbSendToMachines map[Tid]chan *Message, lbRecv chan *Message) *GlobalSched {
	lb := &GlobalSched{
		machines:         machines,
		procq:            &Queue{q: make([]*Proc, 0)},
		machineConnRecv:  lbRecv,
		machineConnSend:  lbSendToMachines,
		currTick:         0,
		procTypeProfiles: make(map[ProcType]*ProvProcDistribution),
	}

	// hard-coded for now
	lb.procTypeProfiles[PAGE_STATIC] = &ProvProcDistribution{
		computeUsed: Distribution{avg: float64(PAGE_STATIC_SLA) - PAGE_STATIC.getExpectedSlaBuffer()*float64(PAGE_STATIC_SLA), count: 0,
			stdDev: PAGE_STATIC.getExpectedProcDeviationVariance()},
		memUsg: Distribution{avg: PAGE_STATIC_MEM_USG, count: 0, stdDev: 0},
	}
	lb.procTypeProfiles[PAGE_DYNAMIC] = &ProvProcDistribution{
		computeUsed: Distribution{avg: float64(PAGE_DYNAMIC_SLA) - PAGE_DYNAMIC.getExpectedSlaBuffer()*float64(PAGE_DYNAMIC_SLA), count: 0,
			stdDev: PAGE_DYNAMIC.getExpectedProcDeviationVariance()},
		memUsg: Distribution{avg: PAGE_DYNAMIC_MEM_USG, count: 0, stdDev: 0},
	}
	lb.procTypeProfiles[DATA_PROCESS_FG] = &ProvProcDistribution{
		computeUsed: Distribution{avg: float64(DATA_PROCESS_FG_SLA) - DATA_PROCESS_FG.getExpectedSlaBuffer()*float64(DATA_PROCESS_FG_SLA), count: 0,
			stdDev: DATA_PROCESS_FG.getExpectedProcDeviationVariance()},
		memUsg: Distribution{avg: DATA_PROCESS_FG_MEM_USG, count: 0, stdDev: 0},
	}
	lb.procTypeProfiles[DATA_PROCESS_BG] = &ProvProcDistribution{
		computeUsed: Distribution{avg: float64(DATA_PROCESS_BG_SLA) - DATA_PROCESS_BG.getExpectedSlaBuffer()*float64(DATA_PROCESS_BG_SLA), count: 0,
			stdDev: DATA_PROCESS_BG.getExpectedProcDeviationVariance()},
		memUsg: Distribution{avg: DATA_PROCESS_BG_MEM_USG, count: 0, stdDev: 0},
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
				toWrite := fmt.Sprintf("%v, %v, %v, %v, %v, %v\n", lb.currTick, msg.proc.machineId, msg.proc.procInternals.procType, float64(msg.proc.procInternals.sla), float64(msg.proc.ticksPassed), float64(msg.proc.procInternals.actualComp))
				logWrite(DONE_PROCS, toWrite)
			}
			if _, ok := lb.procTypeProfiles[msg.proc.procType()]; ok {
				lb.procTypeProfiles[msg.proc.procType()].updateMem(msg.proc.memUsed())
				lb.procTypeProfiles[msg.proc.procType()].updateCompute(msg.proc.compUsed())
			} else {
				lb.procTypeProfiles[msg.proc.procType()] = newProcProcDistribution(msg.proc.memUsed(), msg.proc.compUsed())
			}
		}
	}
}

func (lb *GlobalSched) placeProcs() {
	// setup
	lb.currTick += 1
	p := lb.getProc()

	for p != nil {
		// place given proc

		machineToUse := lb.pickMachine(p)

		// place proc on chosen machine
		p.machineId = machineToUse.mid
		p.procTypeProfile = lb.procTypeProfiles[p.procType()]
		var wg sync.WaitGroup
		wg.Add(1)
		lb.machineConnSend[machineToUse.mid] <- &Message{-1, LB_M_PLACE_PROC, p, &wg}
		wg.Wait()
		if VERBOSE_LB_STATS {
			toWrite := fmt.Sprintf("%v, %v, %v, %v, %v\n", lb.currTick, machineToUse.mid, p.procInternals.procType, float64(p.procInternals.sla), float64(p.procInternals.actualComp))
			logWrite(ADDED_PROCS, toWrite)
		}
		p = lb.getProc()
	}
}

// admission control:
// 1. will the machine be able to handle the proc? (in terms of cpu time as well as memory)
// 2. among those that are, which one has the lowest load?
func (lb *GlobalSched) pickMachine(procToPlace *Proc) *Machine {

	var machineToUse *Machine
	contenderMachines := make([]*Machine, 0)

	for _, m := range lb.machines {
		// TODO: is this how we want to do it? (asking every time); makes it so we can't reuse the probes...
		if m.sched.memFree() > float64(procToPlace.expectedMem) && m.sched.okToPlace(procToPlace) {
			contenderMachines = append(contenderMachines, m)
		}
	}

	toWrite := fmt.Sprintf("%v, LB placing proc: %v, there are %v contender machines \n", lb.currTick, procToPlace.String(), len(contenderMachines))
	logWrite(SCHED, toWrite)

	if len(contenderMachines) == 0 {
		fmt.Println("DOESN'T FIT ANYWHERE :((")

		os.Exit(0)
	}

	minVal := math.Inf(0)
	for _, m := range contenderMachines {
		if m.sched.maxRatioTicksPassedToSla() < minVal {
			machineToUse = m
			minVal = m.sched.maxRatioTicksPassedToSla()
		}
	}

	return machineToUse
}

func (lb *GlobalSched) getProc() *Proc {
	return lb.procq.deq()
}

func (lb *GlobalSched) putProc(proc *Proc) {
	lb.procq.enq(proc)
}
