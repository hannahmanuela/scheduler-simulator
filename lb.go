package slasched

import (
	"fmt"
	"math"
	"slices"
	"sync"

	"golang.org/x/exp/maps"
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
	machinesNotInUse   []*Machine
	procq              *Queue
	machineConn        chan *MachineMessages
	numProcsKilled     int
	numProcsOverSLA_TN int // true negatives, ie ones who could have been completed in their sla
	numProcsOverSLA_FN int // fals enegatives, ie ones whose compute time was longer that the sla
	currTick           int
	printedThisTick    bool
}

func newLoadBalancer(machines []*Machine, machineConn chan *MachineMessages) *LoadBalancer {
	lb := &LoadBalancer{
		// start with one machine?
		machines:         []*Machine{machines[0]},
		machinesNotInUse: machines[1:],
		// machines:         machines,
		// machinesNotInUse: []*Machine{},
		procq:           &Queue{q: make([]*Proc, 0)},
		machineConn:     machineConn,
		currTick:        0,
		numProcsKilled:  0,
		printedThisTick: false,
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
			//  when a proc is done, the ticksPassed on it is updated to be exact, so we don't have to worry about half ticks here
			if msg.proc.timeLeftOnSLA() < 0 {
				// proc went over based on sla, but was it over given actual compute?
				// floats are weird just deal with it
				if math.Abs(float64(msg.proc.ticksPassed-msg.proc.procInternals.actualComp)) > 0.000001 {
					// yes, even actual compute was less than ticks passed
					lb.numProcsOverSLA_TN += 1
					if VERBOSE_LB_STATS {
						fmt.Printf("done: %v, %v, %v, %v, %v, %v, 1 \n", lb.currTick, msg.proc.machineId, msg.proc.procInternals.procType, float64(msg.proc.procInternals.sla), float64(msg.proc.ticksPassed), float64(msg.proc.procInternals.actualComp))
					}
				} else {
					// no, was in fact impossible to get it done on time (b/c we did the very best we could, ie ticksPassed = actualComp)
					lb.numProcsOverSLA_FN += 1
					if VERBOSE_LB_STATS {
						fmt.Printf("done: %v, %v, %v, %v, %v, %v, 0 \n", lb.currTick, msg.proc.machineId, msg.proc.procInternals.procType, float64(msg.proc.procInternals.sla), float64(msg.proc.ticksPassed), float64(msg.proc.procInternals.actualComp))
					}
				}
			}
		case PROC_KILLED:
			if VERBOSE_LB_STATS {
				fmt.Printf("killing: %v, %v, %v, %v, %v\n", lb.currTick, msg.proc.machineId, float64(msg.proc.procInternals.sla), float64(msg.proc.procInternals.compDone), float64(msg.proc.procInternals.memUsed))
			}
			lb.numProcsKilled += 1
			lb.procq.enq(msg.proc)
			// msg.wg.Done()
		}
	}
}

func (lb *LoadBalancer) placeProcs() {
	// setup
	lb.printedThisTick = false
	lb.currTick += 1
	p := lb.getProc()

	// decide if we should add/remove a machine -- NOTE that we currently only add max one machine
	totalProcs, memUsg, numProcsKilled := lb.getMachineStats()
	// decrease if values are low and we have multiple machines; use AND
	// increase if values are high and we have machines we can add; use OR
	if avg(maps.Values(memUsg)) < THRESHOLD_MEM_USG_MIN && avg(maps.Values(totalProcs)) < THRESHOLD_NUM_PROCS_MIN && len(lb.machines) > 1 {
		toRemove := lb.machines[0]
		lb.machines = lb.machines[1:]
		lb.machinesNotInUse = append(lb.machinesNotInUse, toRemove)
		if VERBOSE_SCHED_STATS {
			fmt.Printf("machine: 0, %v, %v, %v, %v\n", lb.currTick, toRemove.mid, avg(maps.Values(memUsg)), avg(maps.Values(totalProcs)))
		}
	} else if (numProcsKilled > 0 || avg(maps.Values(memUsg)) > THRESHOLD_MEM_USG_MAX || avg(maps.Values(totalProcs)) > THRESHOLD_NUM_PROCS_MAX) && len(lb.machinesNotInUse) > 0 {
		toAdd := lb.machinesNotInUse[0]
		lb.machinesNotInUse = lb.machinesNotInUse[1:]
		lb.machines = append(lb.machines, toAdd)
		if VERBOSE_SCHED_STATS {
			fmt.Printf("machine: 1, %v, %v, %v, %v, %v\n", lb.currTick, toAdd.mid, avg(maps.Values(memUsg)), avg(maps.Values(totalProcs)), numProcsKilled)
		}
	}

	for p != nil {
		// place given proc
		if VERBOSE_LB {
			fmt.Printf("placing proc %v\n", p)
		}

		procsInRange := lb.getMachineStatsForRange(p.procInternals.sla)

		machineWeights := lb.calcluateWeights(procsInRange)

		// place proc on machine
		machineToUse := lb.machines[findMaxIndex(machineWeights)]
		p.machineId = machineToUse.mid
		machineToUse.sched.q.enq(p)
		if VERBOSE_LB_STATS {
			if p.migrated {
				fmt.Printf("adding: %v, %v, %v, %v, %v, 1\n", lb.currTick, machineToUse.mid, p.procInternals.procType, float64(p.procInternals.sla), float64(p.procInternals.actualComp))
			} else {
				fmt.Printf("adding: %v, %v, %v, %v, %v, 0\n", lb.currTick, machineToUse.mid, p.procInternals.procType, float64(p.procInternals.sla), float64(p.procInternals.actualComp))
			}

		}
		p = lb.getProc()
	}
}

// DIFFERENT SCHEDULING GOALS IN REL TO SIGNALS WE HAVE

// goal 1: maintain a similar distribution across all the machines for different SLAs
// 	- benefit: no one machine will have all the procs with tight SLAs, but they are spread out (so that if they end up needing more than expected, there's few other small procs that have already been placed and are waiting, mostly procs that have a long SLA anyway)
// 	- drawback: what if procs that have similar SLAs also share a bunch of stuff (state etc)
// 			--> what if we also look at that, treat it as a separate dimension rather than using SLA as a proxy (what would the signal here be? cld look at what function the proc came from?)

// goal 2: don't place proc on machine where mem is tight
// - the higher a machine's mem pressure is, the less procs we should place there
func (lb *LoadBalancer) calcluateWeights(procsInRange map[*Machine]int) []float64 {

	maxNumProcsInRange := slices.Max(maps.Values(procsInRange))

	var machineWeights []float64
	for _, m := range lb.machines {
		// memory factor
		memFree := float64(MAX_MEM - m.sched.memUsed())
		weight := memFree
		// tke into account num procs in range of placed proc if there are others already
		if maxNumProcsInRange > 0 {
			numProcsInRange := procsInRange[m]
			diffToMaxNumProcs := maxNumProcsInRange - numProcsInRange
			// MAX_MEM is going to be the maximal value possible (so that its equally weighted with mem - FOR NOW - )
			normedDiffToMaxNumProcs := float64(diffToMaxNumProcs) * (float64(MAX_MEM) / float64(maxNumProcsInRange))
			weight += normedDiffToMaxNumProcs
			if VERBOSE_LB {
				fmt.Printf("given that the max num procs in this range is %v, and this machine has %v procs (diff: %v, normed: %v), gave it weight %v\n", maxNumProcsInRange, numProcsInRange, diffToMaxNumProcs, normedDiffToMaxNumProcs, weight)
			}
		} else {
			if VERBOSE_LB {
				fmt.Printf("no procs in this range yet\n")
			}
		}
		machineWeights = append(machineWeights, weight)
	}

	return machineWeights
}

// returns:
// totalProcs: number of procs on each machine
// memUsg: the memory usage of each machine
func (lb *LoadBalancer) getMachineStats() (map[*Machine]int, map[*Machine]float64, int) {
	numProcsKilled := 0
	totalProcs := make(map[*Machine]int, 0)
	memUsg := make(map[*Machine]float64, 0)
	for _, m := range lb.machines {
		totalProcs[m] = m.sched.q.qlen()
		memUsg[m] = m.sched.memUsage()
		numProcsKilled += m.sched.numProcsKilledLastTick
	}

	return totalProcs, memUsg, numProcsKilled
}

// returns:
// procsInRange: the number of procs in the same range as the given sla per machine
func (lb *LoadBalancer) getMachineStatsForRange(sla Tftick) map[*Machine]int {
	procsInRange := make(map[*Machine]int, 0)
	for _, m := range lb.machines {
		// I could also do this by just being able to get the number of procs on a machine within a certain range
		histogram := m.sched.makeHistogram()
		numProcsInRange := histogram[m.sched.getRangeBottomFromSLA(sla)]
		procsInRange[m] = numProcsInRange
	}

	return procsInRange
}

func (lb *LoadBalancer) getProc() *Proc {
	return lb.procq.deq()
}

func (lb *LoadBalancer) putProc(proc *Proc) {
	lb.procq.enq(proc)
}
