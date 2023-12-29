package slasched

import (
	"fmt"
	"math"
)

const (
	EWMA_ALPHA = 0.2
)

type Schedd struct {
	totMem   Tmem
	q        *Queue
	pressure float64 // average left over budget (using an ewma with alpha value EWMA_ALPHA)
	numCores int
}

func newSchedd(numCores int) *Schedd {
	sd := &Schedd{
		totMem:   MAX_MEM,
		q:        newQueue(),
		pressure: 1,
		numCores: numCores,
	}
	return sd
}

func (sd *Schedd) String() string {
	str := fmt.Sprintf("{pressure: %v, ", sd.pressure)
	str += "q: \n"
	for _, p := range sd.q.q {
		str += "    " + p.String() + "\n"
	}
	str += "}"
	return str
}

func (sd *Schedd) memUsed() Tmem {
	m := Tmem(0)
	for _, p := range sd.q.q {
		m += p.memUsed()
	}
	return m
}

func (sd *Schedd) tick() {
	if len(sd.q.q) == 0 {
		return
	}
	sd.runProcs()
	for _, currProc := range sd.q.q {
		currProc.ticksPassed += 1
	}
}

// do 1 tick of computation, spread across procs in q, and across different cores
func (sd *Schedd) runProcs() {
	ticksLeftToGive := Tftick(1)

	for ticksLeftToGive > 0 && sd.q.qlen() > 0 {

		ticksPerProc := sd.allocTicksToProc(ticksLeftToGive)
		newProcQ := make([]*ProvProc, 0)
	PROCLOOP:
		for _, currProc := range sd.q.q {
			allocatedComp := ticksPerProc[currProc]
			if allocatedComp.cores == 0 || allocatedComp.time == 0 {
				fmt.Println("idle proc, skipping")
				newProcQ = append(newProcQ, currProc)
				continue PROCLOOP
			}
			ticksUsed, done := currProc.runTillOutOrDone(allocatedComp)
			fmt.Printf("used %v ticks\n", ticksUsed)
			if !done {
				newProcQ = append(newProcQ, currProc)
			}
			ticksLeftToGive -= ticksUsed
			if done {
				// do this not just when proc is done but every iter? slightly changes the point of the proc
				diffToSLA := currProc.timeLeftOnSLA() - (1 - ticksLeftToGive)
				sd.updatePressure(diffToSLA)
			}
		}

		sd.q.q = newProcQ

		if sd.q.qlen() > 0 {
			if ticksLeftToGive > 0.001 {
				fmt.Printf("another round of scheduling %v\n", ticksLeftToGive)
			} else {
				ticksLeftToGive = Tftick(0)
			}
		}

	}
}

// optimistic
// TODO: this whole thing will become more complicated if I start giving procs
// fractional ticks on fractions of the machines cores -- for now always giving all cores
func (sd *Schedd) allocTicksToProc(ticksLeftToGive Tftick) map[*ProvProc]computeTime {

	procToTicks := make(map[*ProvProc]computeTime, 0)

	ticksGiven := Tftick(0)
	for _, currProc := range sd.q.q {
		allocatedCores := sd.numCores
		allocatedTicks := Tftick(math.Min(float64(currProc.timeLeftOnSLA()), float64(ticksLeftToGive-ticksGiven)))

		// panic.
		if currProc.timeLeftOnSLA() < 0 {
			allocatedTicks = Tftick(ticksLeftToGive - ticksGiven)
		}

		fmt.Printf("giving %v ticks, and %d cores \n", allocatedTicks, allocatedCores)
		procToTicks[currProc] = computeTime{allocatedTicks, allocatedCores}

		ticksGiven += allocatedTicks
	}

	return procToTicks
}

// currently just using EWMA, in both cases (went over SLA and was still under)
func (sd *Schedd) updatePressure(diffToSLA Tftick) {
	fmt.Printf("updating pressure given diff: %v \n", diffToSLA)
	sd.pressure = EWMA_ALPHA*float64(diffToSLA) + (1-EWMA_ALPHA)*sd.pressure
}
