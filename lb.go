package slasched

import (
	"fmt"
	"sync"
)

type MachineMsgType int // messages for the machine, by the load balancer

const (
	ADD_PROC MachineMsgType = iota
)

type MachineMessages struct {
	msgType MachineMsgType
	proc    *Proc
	wg      *sync.WaitGroup
}

type LoadBalancer struct {
	machines           []*Machine
	procq              *Queue
	machineConn        chan *MachineMessages
	nextMachineForWork int
}

func newLoadBalancer(machines []*Machine, machineConn chan *MachineMessages) *LoadBalancer {
	lb := &LoadBalancer{
		machines:           machines,
		procq:              &Queue{q: make([]*Proc, 0)},
		machineConn:        machineConn,
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
	p := lb.getProc()

	for p != nil {
		// place given proc
		if VERBOSE_LB {
			fmt.Printf("placing proc %v\n", p)
		}

		machineToUse := lb.machines[lb.nextMachineForWork]

		// place proc on chosen machine
		p.machineId = machineToUse.mid
		var wg sync.WaitGroup
		wg.Add(1)
		machineToUse.sched.lbConn <- &MachineMessages{ADD_PROC, p, &wg}
		wg.Wait()
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
