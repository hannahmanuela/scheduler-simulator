package slasched

import (
	"fmt"
	"math"
)

type MsgType int

const (
	PROC_DONE MsgType = iota
	PROC_KILLED
)

type MachineMessages struct {
	msgType MsgType
	proc    *Proc
}

type LoadBalancer struct {
	machines           []*Machine
	procq              *Queue
	machineConn        chan *MachineMessages
	numProcsKilled     int
	numProcsOverSLA_TN int // true negatives, ie ones who could have been completed in their sla
	numProcsOverSLA_FN int // fals enegatives, ie ones whose compute time was longer that the sla
}

func newLoadBalancer(machines []*Machine, machineConn chan *MachineMessages) *LoadBalancer {
	lb := &LoadBalancer{machines: machines}
	lb.procq = &Queue{q: make([]*Proc, 0)}
	lb.machineConn = machineConn
	lb.numProcsKilled = 0
	go lb.listenForMachineMessages()
	return lb
}

func (lb *LoadBalancer) listenForMachineMessages() {
	for {
		msg := <-lb.machineConn
		switch msg.msgType {
		case PROC_DONE:
			//  when a proc is done, the ticksPassed on it is updated to be exact, so we don't have to worry about half ticks here
			if msg.proc.timeLeftOnSLA() < 0 {
				// proc went over based on sla, but was it over given actual compute?
				// floats are weird just deal with it
				if math.Abs(float64(msg.proc.ticksPassed-msg.proc.procInternals.actualComp)) > 0.000001 {
					// yes, even actual compute was less than ticks passed
					if VERBOSE_WORLD {
						fmt.Printf("lb received done proc, was a true negative: ticksPassed: %v, sla: %v, actual compute: %v\n", msg.proc.ticksPassed, msg.proc.procInternals.sla, msg.proc.procInternals.actualComp)
					}
					lb.numProcsOverSLA_TN += 1
				} else {
					// no, was in fact impossible to get it done on time (b/c we did the very best we could, ie ticksPassed = actualComp)
					if VERBOSE_WORLD {
						fmt.Printf("lb received done proc, was a false negative: ticksPassed: %v, sla: %v, actual compute: %v\n", msg.proc.ticksPassed, msg.proc.procInternals.sla, msg.proc.procInternals.actualComp)
					}
					lb.numProcsOverSLA_FN += 1
				}
			}
		case PROC_KILLED:
			if VERBOSE_WORLD {
				fmt.Printf("lb received killed proc, requeuing\n")
			}
			lb.numProcsKilled += 1
			lb.procq.enq(msg.proc)
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
					fmt.Printf("given that the max num procs in this range is %v, and this machine has %v procs (diff: %v, normed: %v), and %v mem free, gave it weight %v\n", maxValInRange, numProcsInRange, diffToMaxNumProcs, normedDiffToMaxNumProcs, memFree, weight)
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
		machineToUse.sched.q.enq(p)
		p = lb.getProc()
	}
}

func (lb *LoadBalancer) getProc() *Proc {
	return lb.procq.deq()
}

func (lb *LoadBalancer) putProc(proc *Proc) {
	lb.procq.enq(proc)
}
