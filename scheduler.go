package slasched

import (
	"fmt"
	"sort"
)

const (
	EWMA_ALPHA = 0.2
)

type Sched struct {
	totMem   Tmem
	q        *Queue
	pressure float64 // average left over budget (using an ewma with alpha value EWMA_ALPHA)
}

func newSched() *Sched {
	sd := &Sched{
		totMem:   MAX_MEM,
		q:        newQueue(),
		pressure: 0,
	}
	return sd
}

func (sd *Sched) String() string {
	str := fmt.Sprintf("{pressure: %v, mem usage: %v, ", sd.pressure, sd.memUsage())
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

func (sd *Sched) tick() {
	if len(sd.q.q) == 0 {
		return
	}
	sd.runProcs()
	for _, currProc := range sd.q.q {
		currProc.ticksPassed += 1
	}
}

// do 1 tick of computation, spread across procs in q, and across different cores
// TODO: use enq and deq here? otherwise structure in q is not actually maintained
func (sd *Sched) runProcs() {
	ticksLeftToGive := Tftick(1)

OUTERLOOP:
	for ticksLeftToGive > 0.001 && sd.q.qlen() > 0 {

		ticksPerProc := sd.allocTicksToProcs(ticksLeftToGive)
		newProcQ := make([]*Proc, 0)
	PROCLOOP:
		for idx, currProc := range sd.q.q {
			allocatedComp := ticksPerProc[currProc]
			if allocatedComp == 0 {
				fmt.Println("idle proc, skipping")
				newProcQ = append(newProcQ, currProc)
				continue PROCLOOP
			}
			ticksUsed, done := currProc.runTillOutOrDone(allocatedComp)
			ticksLeftToGive -= ticksUsed
			fmt.Printf("used %v ticks\n", ticksUsed)
			if !done {
				newProcQ = append(newProcQ, currProc)
				if sd.memUsed() >= sd.totMem {
					fmt.Println("----- KILLING -----")
					sd.kill(idx, newProcQ)
					continue OUTERLOOP
				}
			} else {
				// do this not just when proc is done but every iter? slightly changes the point of the proc
				diffToSLA := currProc.timeLeftOnSLA() - (1 - ticksLeftToGive)
				sd.updatePressure(diffToSLA)
			}
		}

		sd.q.q = newProcQ
	}
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
		fmt.Printf("giving %v ticks \n", allocatedTicks)
		procToTicks[currProc] = allocatedTicks
		ticksGiven += allocatedTicks
	}

	return procToTicks
}

// currently just using EWMA, in both cases (went over SLA and was still under)
func (sd *Sched) updatePressure(diffToSLA Tftick) {
	fmt.Printf("updating pressure given diff: %v \n", diffToSLA)
	sd.pressure = EWMA_ALPHA*float64(diffToSLA) + (1-EWMA_ALPHA)*sd.pressure
}

func (sd *Sched) kill(currProcIdx int, newProcQ []*Proc) {

	currQueue := append(newProcQ, sd.q.q[:currProcIdx]...)

	// sort by killable score :D
	sort.Slice(currQueue, func(i, j int) bool {
		return currQueue[i].killableScore() < currQueue[j].killableScore()
	})

	memCut := 0

	// this threshold is kinda arbitrary
	// TODO: tell main scheduler that this proc has been killed
	for memCut < 2 {
		killed := currQueue[0]
		currQueue = currQueue[1:]
		memCut += int(killed.memUsed())
		fmt.Printf("killing proc %s gave us back %d memory\n", killed.String(), memCut)
	}

	sd.q.q = currQueue
}
