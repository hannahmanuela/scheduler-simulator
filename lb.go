package slasched

import (
	"fmt"
)

type MachineMsgType int // messages for the machine, by the load balancer

const (
	ADD_PROC MachineMsgType = iota
)

type MachineMessages struct {
	msgType MachineMsgType
	proc    *Proc
}

type LoadBalancer struct {
	machines           []*Machine
	procq              *Queue
	machineConn        chan *MachineMessages
	currTick           int
	nextMachineForWork int
}

func newLoadBalancer(machines []*Machine, machineConn chan *MachineMessages) *LoadBalancer {
	lb := &LoadBalancer{
		machines:           machines,
		procq:              &Queue{q: make([]*Proc, 0)},
		machineConn:        machineConn,
		currTick:           0,
		nextMachineForWork: 0,
	}
	return lb
}

func (lb *LoadBalancer) MachinesString() string {
	str := "machines: \n"
	for _, m := range lb.machines {
		str += "   " + m.String()
	}
	return str
}

func (lb *LoadBalancer) placeProcs() {

	// setup
	lb.currTick += 1
	p := lb.getProc()

	for p != nil {
		// place given proc
		if VERBOSE_LB {
			fmt.Printf("placing proc %v\n", p)
		}

		machineToUse := lb.machines[lb.nextMachineForWork]

		// place proc on chosen machine
		p.machineId = machineToUse.mid
		// TODO: this needs to be safe, should probably be a msg to the dispatcher
		machineToUse.sched.lbConn <- &MachineMessages{ADD_PROC, p}
		if VERBOSE_LB_STATS {
			if p.migrated {
				fmt.Printf("adding: %v, %v, %v, %v, %v, 1\n", lb.currTick, machineToUse.mid, p.procInternals.procType, float64(p.procInternals.sla), float64(p.procInternals.actualComp))
			} else {
				fmt.Printf("adding: %v, %v, %v, %v, %v, 0\n", lb.currTick, machineToUse.mid, p.procInternals.procType, float64(p.procInternals.sla), float64(p.procInternals.actualComp))
			}

		}
		p = lb.getProc()
		lb.nextMachineForWork = (lb.nextMachineForWork + 1) % len(lb.machines)
	}
}

func (lb *LoadBalancer) getProc() *Proc {
	return lb.procq.deq()
}

func (lb *LoadBalancer) putProc(proc *Proc) {
	lb.procq.enq(proc)
}
