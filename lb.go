package slasched

import "math"

type LoadBalancer struct {
	machines map[Tmid]*Machine
	procq    *Queue
}

func newLoadBalancer(machines map[Tmid]*Machine) *LoadBalancer {
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

// do we put this in the scheduler? we might want to diff between the schedulers (load balancer vs scheduler or something)
// rather than using directly via values, could also sample with probability
func (lb *LoadBalancer) placeProcs() {
	p := lb.getProc()
	for p != nil {
		// place given proc
		var machineToUse *Machine
		minRangeVal := math.Inf(1)
		// TODO: have this balance number of procs with memory pressure
		for _, m := range lb.machines {
			// I could also do this by just being able to get the number of procs on a machine within a certain range
			histogram := m.sched.makeHistogram()
			rangeVal := float64(histogram[m.sched.getRangeBottomFromSLA(p.procInternals.sla)])
			if rangeVal < minRangeVal {
				machineToUse = m
				minRangeVal = rangeVal
			}
		}
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
