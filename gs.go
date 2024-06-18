package slasched

import (
	"fmt"
	"math"
	"math/rand"
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

		// keep profiles on procs, use that
		// sample machines and see which one might be good
		var machineToUse *Machine
		if _, ok := lb.procTypeProfiles[p.procType()]; ok {
			// if we have profiling information, use it
			machineToUse = lb.pickMachineGivenProfile(p)
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
func (lb *GlobalSched) pickMachineGivenProfile(procToPlace *Proc) *Machine {

	profile := lb.procTypeProfiles[procToPlace.procType()]

	toWrite := fmt.Sprintf("%v, LB placing proc: %v \n", lb.currTick, procToPlace.effectiveSla())
	logWrite(SCHED, toWrite)

	minProcsInRange := int(math.Inf(1))
	maxProcsInRange := 0
	minMaxRatioTicksPassedToSla := math.Inf(0)
	maxMaxRatioTicksPassedToSla := 0.0
	minExpectedCompInQ := math.Inf(0)
	maxExpectedCompInQ := 0.0
	for _, m := range lb.machines {
		// machine is a contender if has memory for it
		if m.sched.memFree() > profile.memUsg.avg+profile.memUsg.stdDev {
			if m.sched.maxRatioTicksPassedToSla() > maxMaxRatioTicksPassedToSla {
				maxMaxRatioTicksPassedToSla = m.sched.maxRatioTicksPassedToSla()
			}
			if m.sched.maxRatioTicksPassedToSla() < minMaxRatioTicksPassedToSla {
				minMaxRatioTicksPassedToSla = m.sched.maxRatioTicksPassedToSla()
			}
			if m.sched.procsInRange(procToPlace.effectiveSla()) > maxProcsInRange {
				maxProcsInRange = m.sched.procsInRange(procToPlace.effectiveSla())
			}
			if m.sched.procsInRange(procToPlace.effectiveSla()) < minProcsInRange {
				minProcsInRange = m.sched.procsInRange(procToPlace.effectiveSla())
			}
			if m.sched.expectedCompInQ() > maxExpectedCompInQ {
				maxExpectedCompInQ = m.sched.expectedCompInQ()
			}
			if m.sched.expectedCompInQ() < minExpectedCompInQ {
				minExpectedCompInQ = m.sched.expectedCompInQ()
			}
		}
	}
	// toWrite = fmt.Sprintf("minProcsInRange: %v, maxProcsInRange: %v, minTicksInQ: %v, maxTicksInQ: %v, minRatioSlaToTicksPassed: %v, maxRatioSlaToTicksPassed: %v \n", minProcsInRange, maxProcsInRange, minTicksInQ, maxTicksInQ, minRatioSlaToTicksPassed, maxRatioSlaToTicksPassed)
	// logWrite(SCHED, toWrite)

	machineToPressure := make(map[*Machine]float64, 0)
	for _, m := range lb.machines {
		// machine is a contender if has memory for it
		if m.sched.memFree() > profile.memUsg.avg+profile.memUsg.stdDev {
			// factors: num procs in range; min max sla to ticksPassed ratio [for both, being smaller is better]
			// normalized based on above min/max values

			// TODO: do this by rounds of excluding?
			press := math.Inf(1)
			if m.sched.procsInRange(procToPlace.effectiveSla()) == minProcsInRange {
				if maxExpectedCompInQ != minExpectedCompInQ {
					press = float64((m.sched.expectedCompInQ())-minExpectedCompInQ) / float64(maxExpectedCompInQ-minExpectedCompInQ)
				} else {
					// case where they all have exactly the same procs in range and ticks in q
					press = 0
				}
			}
			machineToPressure[m] = press
			if VERBOSE_PRESSURE_VALS {
				toWrite := fmt.Sprintf("giving machine %v pressure val %v, with a maxRatio of %v, procsInRange of %v, and tikcsInQ of %v \n",
					m.mid, press, m.sched.maxRatioTicksPassedToSla(), m.sched.procsInRange(procToPlace.effectiveSla()), m.sched.ticksInQ())
				logWrite(SCHED, toWrite)
			}
		}
	}

	// TODO: what if no machines are contenders because no one has the memory for the new proc?
	// have a q on the lb?
	if len(machineToPressure) == 0 {
		fmt.Println("EVERYONE OOM")
		os.Exit(0)
	}

	var machineToUse *Machine
	minPressure := math.Inf(1)
	for machine, press := range machineToPressure {
		if press < minPressure {
			machineToUse = machine
			minPressure = press
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
