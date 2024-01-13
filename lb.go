package slasched

type LoadBalancer struct {
	machines []*Machine
	procq    *Queue
}

func newLoadBalancer(machines []*Machine) *LoadBalancer {
	lb := &LoadBalancer{machines: machines}
	lb.procq = &Queue{q: make([]*Proc, 0)}
	return lb
}

// DIFFERENT SCHEDULING GOALS IN REL TO SIGNALS WE HAVE

// goal 1: maintain a similar distribution across all the machines for different SLAs
// 	- benefit: no one machine will have all the procs with tight SLAs, but they are spread out (so that if they end up needing more than expected, there's few other small procs that have already been placed and are waiting, mostly procs that have a long SLA anyway)
// 	- drawback: what if procs that have similar SLAs also share a bunch of stuff (state etc)
// 			--> what if we also look at that, treat it as a separate dimension rather than using SLA as a proxy (what would the signal here be? cld look at what function the proc came from?)

// goal 2: don't place proc on machine where mem is tight
// - the higher a machine's mem pressure is, the less procs we should place there

// rather than using directly via values, could also sample with probability
func (lb *LoadBalancer) placeProcs() {
	p := lb.getProc()
	for p != nil {
		// place given proc
		// fmt.Printf("placing proc %v\n", p)

		// get number of procs in that range on every machine, as well as the max
		rangeVals := make(map[*Machine]int, 0)
		maxVal := 0
		for _, m := range lb.machines {
			// I could also do this by just being able to get the number of procs on a machine within a certain range
			histogram := m.sched.makeHistogram()
			numProcsInRange := histogram[m.sched.getRangeBottomFromSLA(p.procInternals.sla)]
			rangeVals[m] = numProcsInRange
			if numProcsInRange > maxVal {
				maxVal = numProcsInRange
			}
		}

		// calcluate weights
		var machineWeights []float64
		for _, m := range lb.machines {
			memFree := float64(MAX_MEM - m.sched.memUsed())
			weight := memFree
			if maxVal > 0 {
				numProcsInRange := rangeVals[m]
				diffToMaxNumProcs := maxVal - numProcsInRange
				// MAX_MEM is going to be the maximal value possible (so that its equally weighted with mem - FOR NOW - )
				normedDiffToMaxNumProcs := float64(diffToMaxNumProcs) * (float64(MAX_MEM) / float64(maxVal))
				weight += normedDiffToMaxNumProcs
				// fmt.Printf("given that the max num procs in this range is %v, and this machine has %v procs (diff: %v, normed: %v), and %v mem free, gave it weight %v\n", maxVal, numProcsInRange, diffToMaxNumProcs, normedDiffToMaxNumProcs, memFree, weight)
			}
			// else {
			// 	fmt.Printf("no procs in this range yet\n")
			// }
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
