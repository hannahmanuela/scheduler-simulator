package slasched

import (
	"fmt"
	"math"
	"math/rand"
)

type MachineMsgType int

const (
	PROC_DONE MachineMsgType = iota
	PROC_KILLED
)

type CoreMsgType int

const (
	NEED_WORK CoreMsgType = iota // msg by core bc it is out of work
	PUSH_PROC                    // msg by the controller to the core with a proc it should take
	PROC_DONE_CORE
)

type MachineMessages struct {
	msgType MachineMsgType
	proc    *Proc
}

type CoreMessages struct {
	msgType CoreMsgType
	proc    *Proc
}

type LoadBalancer struct {
	machines         map[Tid]*Machine
	procq            *Queue
	machineConn      chan *MachineMessages
	currTick         int
	procTypeProfiles map[ProcType]*ProvProcDistribution
}

func newLoadBalancer(machines map[Tid]*Machine, machineConn chan *MachineMessages) *LoadBalancer {
	lb := &LoadBalancer{
		machines:         machines,
		procq:            &Queue{q: make([]*Proc, 0)},
		machineConn:      machineConn,
		currTick:         0,
		procTypeProfiles: make(map[ProcType]*ProvProcDistribution),
	}
	go lb.listenForMachineMessages()
	return lb
}

func (lb *LoadBalancer) MachinesString() string {
	str := "machines: \n"
	for _, m := range lb.machines {
		str += "   " + m.String()
	}
	return str
}

func (lb *LoadBalancer) listenForMachineMessages() {
	for {
		msg := <-lb.machineConn
		switch msg.msgType {
		case PROC_DONE:
			if VERBOSE_LB_STATS {
				toWrite := fmt.Sprintf("%v, %v, %v, %v, %v, %v, %v\n", lb.currTick, msg.proc.machineId, msg.proc.procInternals.procType, float64(msg.proc.procInternals.sla), float64(msg.proc.ticksPassed), float64(msg.proc.procInternals.actualComp), msg.proc.timesReplenished)
				logWrite(DONE_PROCS, toWrite)
			}
			if _, ok := lb.procTypeProfiles[msg.proc.procType]; ok {
				lb.procTypeProfiles[msg.proc.procType].updateMem(msg.proc.memUsed())
				lb.procTypeProfiles[msg.proc.procType].updateCompute(msg.proc.compUsed())
			} else {
				lb.procTypeProfiles[msg.proc.procType] = newProcProcDistribution(msg.proc.memUsed(), msg.proc.compUsed())
			}
		}
	}
}

func (lb *LoadBalancer) placeProcs() {
	// setup
	lb.currTick += 1
	p := lb.getProc()

	for p != nil {
		// place given proc

		// keep profiles on procs, use that
		// sample machines and see which one might be good
		var machineToUse *Machine
		if _, ok := lb.procTypeProfiles[p.procType]; ok {
			// if we have profiling information, use it
			machineToUse = lb.pickMachineGivenProfile(lb.procTypeProfiles[p.procType])
		} else {
			// otherwise just pick a machine
			machineToUse = lb.machines[Tid(rand.Int()%len(lb.machines))]
		}

		// place proc on chosen machine
		p.machineId = machineToUse.mid
		machineToUse.sched.q.enq(p)
		if VERBOSE_LB_STATS {
			toWrite := fmt.Sprintf("%v, %v, %v, %v, %v\n", lb.currTick, machineToUse.mid, p.procInternals.procType, float64(p.procInternals.sla), float64(p.procInternals.actualComp))
			logWrite(ADDED_PROCS, toWrite)
		}
		p = lb.getProc()
	}
}

// probably actually want this to be via communication with the machine itself, let it say yes or no?
// that way we avoid the "gold rush" things, although since this is one by one anyway maybe its fine
func (lb *LoadBalancer) pickMachineGivenProfile(dist *ProvProcDistribution) *Machine {

	minTicks := math.Inf(1)
	var machineToUse *Machine

	maxMemFree := float64(0)
	var minMemMachine *Machine

	// check if memory fits, and from those machines take the one with the least ticksInQ
	for _, m := range lb.machines {
		if m.sched.memFree() > (dist.memUsg.avg+dist.memUsg.stdDev) && m.sched.ticksInQ() < minTicks {
			minTicks = m.sched.ticksInQ()
			machineToUse = m
		}
		if m.sched.memFree() < maxMemFree {
			maxMemFree = m.sched.memFree()
			minMemMachine = m
		}
	}

	// if memory doesn't fit anywhere, we're fucked I guess? pick machine with least memory used
	if machineToUse == nil {
		fmt.Println("IT DOES'T EVEN FIT ANYWHERE")
		machineToUse = minMemMachine
	}

	return machineToUse
}

func (lb *LoadBalancer) getProc() *Proc {
	return lb.procq.deq()
}

func (lb *LoadBalancer) putProc(proc *Proc) {
	lb.procq.enq(proc)
}
