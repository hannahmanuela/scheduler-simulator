package slasched

import (
	"fmt"
	"math"
	"math/rand"
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

type LoadBalancer struct {
	machines         map[Tid]*Machine
	procq            *Queue
	machineConnRecv  chan *Message         // listen for messages from machines
	machineConnSend  map[Tid]chan *Message // send messages to machine
	currTick         int
	procTypeProfiles map[ProcType]*ProvProcDistribution
}

func newLoadBalancer(machines map[Tid]*Machine, lbSendToMachines map[Tid]chan *Message, lbRecv chan *Message) *LoadBalancer {
	lb := &LoadBalancer{
		machines:         machines,
		procq:            &Queue{q: make([]*Proc, 0)},
		machineConnRecv:  lbRecv,
		machineConnSend:  lbSendToMachines,
		currTick:         0,
		procTypeProfiles: make(map[ProcType]*ProvProcDistribution),
	}

	// hard-coded for now
	lb.procTypeProfiles[PAGE_STATIC] = &ProvProcDistribution{
		computeUsed: Distribution{avg: PAGE_STATIC_SLA_RANGE_MAX, count: 0,
			stdDev: (PAGE_STATIC_SLA_RANGE_MAX - PAGE_STATIC_SLA_RANGE_MIN) / 2.0},
		memUsg: Distribution{avg: PAGE_STATIC_MEM_USG, count: 0, stdDev: 0},
	}
	lb.procTypeProfiles[PAGE_DYNAMIC] = &ProvProcDistribution{
		computeUsed: Distribution{avg: PAGE_DYNAMIC_SLA_RANGE_MAX, count: 0,
			stdDev: (PAGE_DYNAMIC_SLA_RANGE_MAX - PAGE_DYNAMIC_SLA_RANGE_MIN) / 2.0},
		memUsg: Distribution{avg: PAGE_DYNAMIC_MEM_USG, count: 0, stdDev: 0},
	}
	lb.procTypeProfiles[DATA_PROCESS_FG] = &ProvProcDistribution{
		computeUsed: Distribution{avg: DATA_PROCESS_FG_SLA_RANGE_MAX, count: 0,
			stdDev: (DATA_PROCESS_FG_SLA_RANGE_MAX - DATA_PROCESS_FG_SLA_RANGE_MIN) / 2.0},
		memUsg: Distribution{avg: DATA_PROCESS_FG_MEM_USG, count: 0, stdDev: 0},
	}
	lb.procTypeProfiles[DATA_PROCESS_BG] = &ProvProcDistribution{
		computeUsed: Distribution{avg: DATA_PROCESS_BG_SLA_RANGE_MAX, count: 0,
			stdDev: (DATA_PROCESS_BG_SLA_RANGE_MAX - DATA_PROCESS_BG_SLA_RANGE_MIN) / 2.0},
		memUsg: Distribution{avg: DATA_PROCESS_BG_MEM_USG, count: 0, stdDev: 0},
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
		msg := <-lb.machineConnRecv
		switch msg.msgType {
		case M_LB_PROC_DONE:
			if VERBOSE_LB_STATS {
				toWrite := fmt.Sprintf("%v, %v, %v, %v, %v, %v, %v\n", lb.currTick, msg.proc.machineId, msg.proc.procInternals.procType, float64(msg.proc.procInternals.sla), float64(msg.proc.ticksPassed), float64(msg.proc.procInternals.actualComp), msg.proc.timesReplenished)
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

func (lb *LoadBalancer) placeProcs() {
	// setup
	lb.currTick += 1
	p := lb.getProc()

	for p != nil {
		// place given proc

		// keep profiles on procs, use that
		// sample machines and see which one might be good
		var machineToUse *Machine
		if _, ok := lb.procTypeProfiles[p.procType()]; ok {
			// if we have profiling information, use it
			machineToUse = lb.pickMachineGivenProfile(lb.procTypeProfiles[p.procType()])
		} else {
			// otherwise just pick a machine
			machineToUse = lb.machines[Tid(rand.Int()%len(lb.machines))]
		}

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

// probably actually want this to be via communication with the machine itself, let it say yes or no?
// that way we avoid the "gold rush" things, although since this is one by one anyway maybe its fine
func (lb *LoadBalancer) pickMachineGivenProfile(dist *ProvProcDistribution) *Machine {

	// minTicks := math.Inf(1)
	minNumProcs := int(math.Inf(1))
	var machineToUse *Machine

	maxMemFree := float64(0)
	var minMemMachine *Machine

	// check if memory fits, and from those machines take the one with the least ticksInQ
	for _, m := range lb.machines {
		if m.sched.memFree() > (dist.memUsg.avg+dist.memUsg.stdDev) && m.sched.procsInRange(Tftick(dist.computeUsed.avg)) < minNumProcs {
			// minTicks = m.sched.ticksInQ()
			minNumProcs = m.sched.procsInRange(Tftick(dist.computeUsed.avg))
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
