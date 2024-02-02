package slasched

import (
	"fmt"
	"math"
	"sync"
)

type MsgType int

const (
	PROC_DONE MsgType = iota
	PROC_KILLED
)

type MachineMessages struct {
	msgType MsgType
	proc    *Proc
	wg      *sync.WaitGroup
}

type LoadBalancer struct {
	machines           []*Machine
	procq              *Queue
	machineConn        chan *MachineMessages
	numProcsKilled     int
	numProcsOverSLA_TN int // true negatives, ie ones who could have been completed in their sla
	numProcsOverSLA_FN int // fals enegatives, ie ones whose compute time was longer that the sla
	currTick           int
	printedThisTick    bool
	nextMachine        int
}

func newLoadBalancer(machines []*Machine, machineConn chan *MachineMessages) *LoadBalancer {
	lb := &LoadBalancer{machines: machines}
	lb.procq = &Queue{q: make([]*Proc, 0)}
	lb.machineConn = machineConn
	lb.currTick = 0
	lb.numProcsKilled = 0
	lb.printedThisTick = false
	lb.nextMachine = 0
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
			if VERBOSE_STATS {
				fmt.Printf("done: %v, %v, %v, %v, %v, %v \n", lb.currTick, msg.proc.machineId, msg.proc.procInternals.procType, float64(msg.proc.procInternals.sla), float64(msg.proc.ticksPassed), float64(msg.proc.procInternals.actualComp))
			}
			//  when a proc is done, the ticksPassed on it is updated to be exact, so we don't have to worry about half ticks here
			if msg.proc.timeLeftOnSLA() < 0 {
				// proc went over based on sla, but was it over given actual compute?
				// floats are weird just deal with it
				if math.Abs(float64(msg.proc.ticksPassed-msg.proc.procInternals.actualComp)) > 0.000001 {
					// yes, even actual compute was less than ticks passed
					lb.numProcsOverSLA_TN += 1
				} else {
					// no, was in fact impossible to get it done on time (b/c we did the very best we could, ie ticksPassed = actualComp)
					lb.numProcsOverSLA_FN += 1
				}
			}
		case PROC_KILLED:
			if VERBOSE_STATS {
				fmt.Printf("killed: %v, %v, %v, %v, %v\n", lb.currTick, msg.proc.machineId, float64(msg.proc.procInternals.sla), float64(msg.proc.procInternals.compDone), float64(msg.proc.procInternals.memUsed))
			}
			lb.numProcsKilled += 1
			lb.procq.enq(msg.proc)
			msg.wg.Done()
		}
	}
}

// DIFFERENT SCHEDULING GOALS IN REL TO SIGNALS WE HAVE

// goal 1: maintain a similar distribution across all the machines for different SLAs
// 	- benefit: no one machine will have all the procs with tight SLAs, but they are spread out (so that if they end up needing more than expected, there's few other small procs that have already been placed and are waiting, mostly procs that have a long SLA anyway)
// 	- drawback: what if procs that have similar SLAs also share a bunch of stuff (state etc)
// 			--> what if we also look at that, treat it as a separate dimension rather than using SLA as a proxy (what would the signal here be? cld look at what function the proc came from?)

// goal 2: don't place proc on machine where mem is tight
// - the higher a machine's mem pressure is, the less procs we should place there

// goal 3: don't place proc on machine where lots of other procs already are
// - this one should count less, say by a factor of 1/2?

// rather than using directly via values, could also sample with probability
// TODO: why does this not at all account for total number of procs on a machine?
func (lb *LoadBalancer) placeProcs() {
	lb.printedThisTick = false
	lb.currTick += 1
	p := lb.getProc()
	for p != nil {
		// place given proc
		// fmt.Printf("placing proc %v\n", p)

		// get number of procs in the same sla range as the proc to place on every machine, as well as the max
		rangeVals := make(map[*Machine]int, 0)
		maxValInRange := 0
		maxValTotalProcs := 0
		for _, m := range lb.machines {
			// I could also do this by just being able to get the number of procs on a machine within a certain range
			histogram := m.sched.makeHistogram()
			numProcsInRange := histogram[m.sched.getRangeBottomFromSLA(p.procInternals.sla)]
			rangeVals[m] = numProcsInRange
			if numProcsInRange > maxValInRange {
				maxValInRange = numProcsInRange
			}
			if m.sched.q.qlen() > maxValTotalProcs {
				maxValTotalProcs = m.sched.q.qlen()
			}
		}

		// calcluate weights
		var machineWeights []float64
		for _, m := range lb.machines {
			memFree := float64(MAX_MEM - m.sched.memUsed())
			diffToMaxNumProcs := maxValTotalProcs - m.sched.q.qlen()
			normedDiffToMaxNumTotalProcs := float64(diffToMaxNumProcs) * (float64(MAX_MEM) / 2 * float64(maxValTotalProcs)) // factor that this counts less is in the last denominator
			weight := memFree + normedDiffToMaxNumTotalProcs
			if maxValInRange > 0 {
				numProcsInRange := rangeVals[m]
				diffToMaxNumProcs := maxValInRange - numProcsInRange
				// MAX_MEM is going to be the maximal value possible (so that its equally weighted with mem - FOR NOW - )
				normedDiffToMaxNumProcs := float64(diffToMaxNumProcs) * (float64(MAX_MEM) / float64(maxValInRange))
				weight += normedDiffToMaxNumProcs
				if VERBOSE_LB {
					fmt.Printf("given that the max num procs in this range is %v, and this machine has %v procs (diff: %v, normed: %v), gave it weight %v\n", maxValInRange, numProcsInRange, diffToMaxNumProcs, normedDiffToMaxNumProcs, weight)
				}
			} else {
				if VERBOSE_LB {
					fmt.Printf("no procs in this range yet\n")
				}
			}
			machineWeights = append(machineWeights, weight)
		}

		// place proc on machine, chosen by weighted random drawing? (could also just pick max -- trying for now)
		// machineToUse := lb.machines[sampleFromWeightList(machineWeights)]
		machineToUse := lb.machines[findMaxIndex(machineWeights)]
		// machineToUse := lb.machines[lb.nextMachine]
		// lb.nextMachine = (lb.nextMachine + 1) % len(lb.machines)
		p.machineId = machineToUse.mid
		machineToUse.sched.q.enq(p)
		if VERBOSE_STATS {
			fmt.Printf("adding: %v, %v, %v, %v, %v\n", lb.currTick, machineToUse.mid, p.procInternals.procType, float64(p.procInternals.sla), float64(p.procInternals.actualComp))
		}
		p = lb.getProc()
	}
}

func (lb *LoadBalancer) getProc() *Proc {
	return lb.procq.deq()
}

func (lb *LoadBalancer) putProc(proc *Proc) {
	lb.procq.enq(proc)
}
