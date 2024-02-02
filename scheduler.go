package slasched

import (
	"fmt"
	"math"
	"sort"
	"sync"
)

const (
	EWMA_ALPHA = 0.2
)

type Sched struct {
	totMem       Tmem
	q            *Queue
	avgDiffToSla float64 // average left over budget (using an ewma with alpha value EWMA_ALPHA)
	minSliceSize float64 // how little time did the least scheduled proc get
	lbConn       chan *MachineMessages
}

func newSched(lbConn chan *MachineMessages) *Sched {
	sd := &Sched{
		totMem:       MAX_MEM,
		q:            newQueue(),
		avgDiffToSla: 0,
		minSliceSize: 1,
		lbConn:       lbConn,
	}
	return sd
}

func (sd *Sched) String() string {
	str := fmt.Sprintf("{compute pressure: %v, mem usage: %v, ", sd.getComputePressure(), sd.memUsage())
	str += "q: \n"
	for _, p := range sd.q.q {
		str += "    " + p.String() + "\n"
	}
	str += "}"
	return str
}

func (sd *Sched) memUsage() float64 {
	return float64(sd.memUsed()) / float64(sd.totMem)
}

func (sd *Sched) memUsed() Tmem {
	sum := Tmem(0)
	for _, p := range sd.q.q {
		sum += p.memUsed()
	}
	return sum
}

// returns a map from the lower value of the SCHEDULER_SLA_INCREMENT_SIZEd range to the number of procs in that range
// (where a proc being in the range means that the proc has that much time keft before it needs to be done, according to its SLA)
func (sd *Sched) makeHistogram() map[float64]int {

	procMap := make(map[float64]int, 0)

	if sd.q.qlen() == 0 {
		return procMap
	}

	for _, p := range sd.q.q {
		rangeBottom := sd.getRangeBottomFromSLA(p.timeLeftOnSLA())
		procMap[rangeBottom] += 1
	}

	return procMap
}

// returns the bottom value of the SLA range in which the passed SLA is
// this is helpful for creating the histogram mapping number of procs in the scheduler to SLA slices
// eg if we are looking at SLAs in an increment size of 2 and this is given 1.5, it will return 0 (since 1.5 would be in the 0-2 range)
func (sd *Sched) getRangeBottomFromSLA(sla Tftick) float64 {
	val := math.Pow(SCHEDULER_SLA_HISTOGRAM_BASE, math.Floor(math.Log(float64(sla))/math.Log(SCHEDULER_SLA_HISTOGRAM_BASE)))
	return val
}

func (sd *Sched) tick() {
	if len(sd.q.q) == 0 {
		return
	}
	sd.runProcs()
	for _, currProc := range sd.q.q {
		currProc.ticksPassed += 1
	}
}

// this combines the avgDiffToSla of procs completed with the minSliceSize to get a sense of pressure
// one measures how quickly we are able to get smaller procs done, and the other measures how much we are starving longer procs
// this is a slightly weird mixing of different timescales of feedback
func (sd *Sched) getComputePressure() float64 {
	// the smaller the min slice the higher the pressure; slices are fractions of the one tick passed out
	propSliceTakenByOtherProcs := 1.0 - sd.minSliceSize
	// the smaller the avgDiffToSla the higher the pressure (though note it could be negative, which would mean extra high pressure)
	// probably we don't want this to be linear?
	avgTimeWentOverSla := -sd.avgDiffToSla
	// fmt.Printf("getting pressure: propSliceTakenByOtherProcs: %v, avgTimeWentOverSla: %v\n", propSliceTakenByOtherProcs, avgTimeWentOverSla)
	return propSliceTakenByOtherProcs + avgTimeWentOverSla
}

// do 1 tick of computation, spread across procs in q, and across different cores
// TODO: use enq and deq here? otherwise structure in q is not actually maintained -- I'm not sure the structure is actually helpful to us tbh
func (sd *Sched) runProcs() {
	ticksLeftToGive := Tftick(1)
	procToTicksMap := make(map[*Proc]Tftick, 0)

OUTERLOOP:
	for ticksLeftToGive-Tftick(0.001) > 0.0 && sd.q.qlen() > 0 {
		if VERBOSE_SCHEDULER {
			fmt.Printf("scheduling round: ticksLeftToGive is %v, so diff to 0.001 is %v\n", ticksLeftToGive, ticksLeftToGive-Tftick(0.001))
		}
		ticksPerProc := sd.allocTicksToProcs(ticksLeftToGive)
		newProcQ := make([]*Proc, 0)
		for idx, currProc := range sd.q.q {
			// get compute allocated to this proc
			allocatedComp := ticksPerProc[currProc]
			// run the proc
			ticksUsed, done := currProc.runTillOutOrDone(allocatedComp)
			ticksLeftToGive -= ticksUsed
			if VERBOSE_SCHEDULER {
				fmt.Printf("used %v ticks\n", ticksUsed)
			}
			if !done {
				// add ticks used to the tick map - only if the proc isn't done (don't want to take into account small procs that just finished)
				procToTicksMap[currProc] += ticksUsed
				// add proc back into queue, check if the memroy used by the proc sent us over the edge (and if yes, kill then restart)
				newProcQ = append(newProcQ, currProc)
				if sd.memUsed() >= sd.totMem {
					if VERBOSE_SCHEDULER {
						fmt.Println("--> KILLING")
					}
					sd.kill(idx, newProcQ)
					continue OUTERLOOP
				}
			} else {
				// if the proc is done, update the ticksPassed to be exact for metrics etc
				// then update the pressure metric with that value
				currProc.ticksPassed = currProc.ticksPassed + (1 - ticksLeftToGive)
				sd.updateAvgDiffToSLA(currProc.timeLeftOnSLA())
				// don't need to wait if we are just telling it a proc is done
				sd.lbConn <- &MachineMessages{PROC_DONE, currProc, nil}
			}
		}

		sd.q.q = newProcQ
	}

	// update min slice size
	minVal := 1.0
	for _, ticks := range procToTicksMap {
		if ticks < Tftick(minVal) {
			minVal = float64(ticks)
		}
	}
	sd.minSliceSize = minVal

}

// allocates ticks
// inversely proportional to how much time the proc has left on its sla
// if there are procs that are over, will (for now, equally) spread all ticks between them
func (sd *Sched) allocTicksToProcs(ticksLeftToGive Tftick) map[*Proc]Tftick {

	procToTicks := make(map[*Proc]Tftick, 0)

	// get values that allow us to inert the realtionsip between timeLeftOnSLA and ticks given
	// (because more time left should equal less ticks given)
	// also find out if there are procs over the SLA, and if yes how many
	totalTimeLeft := Tftick(0)
	numberOverSLA := 0
	for _, p := range sd.q.q {
		if p.timeLeftOnSLA() <= 0 {
			numberOverSLA += 1
		} else {
			totalTimeLeft += p.timeLeftOnSLA()
		}
	}
	relativeNeedsSum := Tftick(0)
	for _, p := range sd.q.q {
		if p.timeLeftOnSLA() > 0 {
			relativeNeedsSum += totalTimeLeft / p.timeLeftOnSLA()
		}
	}

	ticksGiven := Tftick(0)
	for _, currProc := range sd.q.q {
		allocatedTicks := ((totalTimeLeft / currProc.timeLeftOnSLA()) / relativeNeedsSum) * ticksLeftToGive
		if numberOverSLA > 0 {
			// ~ p a n i c ~
			// go into emergency mode where the tick is only split (for now, evenly) among the procs that are over
			if currProc.timeLeftOnSLA() < 0 {
				allocatedTicks = ticksLeftToGive / Tftick(numberOverSLA)
			} else {
				allocatedTicks = 0
			}
		}
		if VERBOSE_SCHEDULER {
			fmt.Printf("giving %v ticks \n", allocatedTicks)
		}
		procToTicks[currProc] = allocatedTicks
		ticksGiven += allocatedTicks
	}

	return procToTicks
}

// currently just using EWMA, in both cases (went over SLA and was still under)
func (sd *Sched) updateAvgDiffToSLA(diffToSLA Tftick) {
	if VERBOSE_SCHEDULER {
		fmt.Printf("updating pressure given diff: %v \n", diffToSLA)
	}
	sd.avgDiffToSla = EWMA_ALPHA*float64(diffToSLA) + (1-EWMA_ALPHA)*sd.avgDiffToSla
}

func (sd *Sched) kill(currProcIdx int, newProcQ []*Proc) {

	currQueue := append(newProcQ, sd.q.q[currProcIdx+1:]...)
	if VERBOSE_SCHEDULER {
		fmt.Printf("currQ: %v, we are at q index %v\n", currQueue, currProcIdx)
	}

	// sort by killable score :D
	sort.Slice(currQueue, func(i, j int) bool {
		return currQueue[i].killableScore() > currQueue[j].killableScore()
	})

	memCut := 0

	// this threshold is kinda arbitrary
	// TODO: rather than killing, checkpoint and requeue this proc with load balancer?
	for memCut < 2 {
		killed := currQueue[0]
		currQueue = currQueue[1:]
		memCut += int(killed.memUsed())
		var wg sync.WaitGroup
		wg.Add(1)
		sd.lbConn <- &MachineMessages{PROC_KILLED, killed, &wg}
		wg.Wait()
	}

	sd.q.q = currQueue
}
