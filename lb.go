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
			if VERBOSE_LB_STATS {
				fmt.Printf("done: %v, %v, %v, %v, %v, %v, %v\n", lb.currTick, msg.proc.machineId, msg.proc.procInternals.procType, float64(msg.proc.procInternals.sla), float64(msg.proc.ticksPassed), float64(msg.proc.procInternals.actualComp), msg.proc.timesReplenished)
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
			if VERBOSE_LB_STATS {
				fmt.Printf("killing: %v, %v, %v, %v, %v\n", lb.currTick, msg.proc.machineId, float64(msg.proc.procInternals.sla), float64(msg.proc.procInternals.compDone), float64(msg.proc.memUsed()))
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

	for p != nil {
		// place given proc
		if VERBOSE_LB {
			fmt.Printf("placing proc %v\n", p)
		}

		ticksAhead := lb.getMachineStatsForDeadline(p.timeShouldBeDone)
		machineWeights := lb.calcluateWeights(ticksAhead)
		machineToUse := lb.machines[findMaxIndex(machineWeights)]

		// if there is no good option, we add a machine and put it there
		if (machineToUse.sched.getTicksAhead(p.timeShouldBeDone) > THRESHOLD_TICKS_AHEAD_MAX || machineToUse.sched.memUsage() > THRESHOLD_MEM_USG_MAX) && len(lb.machinesNotInUse) > 0 {
			// add a nnew machine
			// NOTE the process of choosing a machine will get more sophisticated later (for now we cycle, ie append to end and pull from front)
			machineToUse = lb.machinesNotInUse[0]
			lb.machinesNotInUse = lb.machinesNotInUse[1:]
			lb.machines = append(lb.machines, machineToUse)
		}

		// place proc on chosen machine
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

	// remove a machine if the load is overall pretty low
	// (should we do this before or after placing this round of procs? after might be a bit overestimating amount of load we are experiencing?
	totalTicks, memUsg := lb.getMachineStats()
	// decrease if values are low and we have multiple machines
	if avg(maps.Values(memUsg)) < THRESHOLD_MEM_USG_MIN && avg(maps.Values(totalTicks)) < THRESHOLD_NUM_TICKS_MIN && len(lb.machines) > 1 {
		// ah, we might want to be a bit more smart in which machine we remove?
		toRemove := lb.machines[0]
		lb.machines = lb.machines[1:]
		lb.machinesNotInUse = append(lb.machinesNotInUse, toRemove)
		if VERBOSE_SCHED_STATS {
			fmt.Printf("machine: 0, %v, %v, %v, %v\n", lb.currTick, toRemove.mid, avg(maps.Values(memUsg)), avg(maps.Values(totalTicks)))
		}
	}
}

// DIFFERENT SCHEDULING GOALS IN REL TO SIGNALS WE HAVE

// goal 1: don't overload on compute
// - if a machine has a lot of work that would have to be done before the current one would be able to run, penalize that machine

// goal 2: don't place proc on machine where mem is tight
// - the higher a machine's mem pressure is, the less procs we should place there
func (lb *LoadBalancer) calcluateWeights(ticksAhead map[*Machine]Tftick) []float64 {

	maxTicksAhead := slices.Max(maps.Values(ticksAhead))

	var machineWeights []float64
	for _, m := range lb.machines {
		// memory factor
		memFree := float64(MAX_MEM - m.sched.memUsed())
		weight := memFree
		// tke into account the amount of ticks a proc would have to wait before it can start to run
		if maxTicksAhead > 0 {
			numTicksAhead := ticksAhead[m]
			diffToMaxTicksAhead := maxTicksAhead - numTicksAhead
			// MAX_MEM is going to be the maximal value possible (so that its equally weighted with mem - FOR NOW - )
			normedDiffToMaxNumTicks := float64(diffToMaxTicksAhead) * (float64(MAX_MEM) / float64(maxTicksAhead))
			weight += normedDiffToMaxNumTicks
			if VERBOSE_LB {
				fmt.Printf("given that the max ticks ahead is %v, and this machine has %v ticks ahead (diff: %v, normed: %v), gave it weight %v\n", maxTicksAhead, numTicksAhead, diffToMaxTicksAhead, normedDiffToMaxNumTicks, weight)
			}
		} else {
			if VERBOSE_LB {
				fmt.Printf("no ticks ahead anywhere!\n")
			}
		}
		machineWeights = append(machineWeights, weight)
	}

	return machineWeights
}

// returns:
// totalTicks: number of expected ticks of computation left
// memUsg: the memory usage of each machine
func (lb *LoadBalancer) getMachineStats() (map[*Machine]Tftick, map[*Machine]float64) {
	totalTicks := make(map[*Machine]Tftick, 0)
	memUsg := make(map[*Machine]float64, 0)
	for _, m := range lb.machines {
		totalTicks[m] = m.sched.q.ticksInQueue()
		memUsg[m] = m.sched.memUsage()
	}

	return totalTicks, memUsg
}

// returns:
// ticksBeforeSla: the sum of the SLA of all procs that have an earlier deadline than the proc being placed (ie how long the proc would have to wait worst case scenario)
func (lb *LoadBalancer) getMachineStatsForDeadline(deadline Tftick) map[*Machine]Tftick {
	ticksAhead := make(map[*Machine]Tftick, 0)
	for _, m := range lb.machines {
		ticks := m.sched.getTicksAhead(deadline)
		ticksAhead[m] = ticks
	}

	return ticksAhead
}

func (lb *LoadBalancer) getProc() *Proc {
	return lb.procq.deq()
}

func (lb *LoadBalancer) putProc(proc *Proc) {
	lb.procq.enq(proc)
}
