package slasched

import (
	"fmt"
	"strconv"
)

// ------------------------------------------------------------------------------------------------
// PROVIDER PROC STRUCT
// ------------------------------------------------------------------------------------------------

// this is the external view of a clients proc, that includes provider-created/maintained metadata, etc
type Proc struct {
	machineId        Tid
	ticksPassed      Tftick
	timeShouldBeDone Tftick
	procInternals    *ProcInternals
	timesReplenished int
	procTypeProfile  *ProvProcDistribution
}

func (p *Proc) String() string {
	return p.procInternals.String() +
		// ", deadline: " + p.timeShouldBeDone.String() +
		", ticks passed: " + p.ticksPassed.String() +
		", times replenished: " + strconv.Itoa(p.timesReplenished)
	// ", procTypeProfile: " + p.procTypeProfile.String()
}

func newProvProc(currTick Ttick, privProc *ProcInternals) *Proc {
	return &Proc{
		machineId:        -1,
		ticksPassed:      0,
		timeShouldBeDone: privProc.sla + Tftick(currTick),
		procInternals:    privProc,
		timesReplenished: 0,
	}
}

// runs proc for the number of ticks passed or until the proc is done,
// returning whether the proc is done and how many ticks were run, as well as whether the proc finished or was forcefully terminated for going over
func (p *Proc) runTillOutOrDone(toRun Tftick) (Tftick, bool, bool) {
	ticksUsed, done := p.procInternals.runTillOutOrDone(toRun)

	// if the proc is running over its sla, double it once, then kill the proc (or should it be allowed to replenish more times than that?)
	if p.procInternals.compDone-p.effectiveSla() >= 0.0 {
		if p.timesReplenished > 0 {
			// pretend the proc is done, even if its not, if we've already replenished it
			return ticksUsed, true, true
		} else {
			p.timesReplenished += 1
			p.timeShouldBeDone += p.procInternals.sla
		}
	}

	return ticksUsed, done, false
}

func (p *Proc) effectiveSla() Tftick {
	return p.procInternals.sla * Tftick(1+p.timesReplenished)
}

func (p *Proc) timeLeftOnSLA() Tftick {
	return p.effectiveSla() - p.ticksPassed
}

func (p *Proc) expectedCompLeft() Tftick {
	return p.effectiveSla() - p.procInternals.compDone
}

func (p *Proc) memUsed() Tmem {
	return p.procInternals.memUsed()
}

func (p *Proc) compUsed() Tftick {
	return p.procInternals.compDone
}

func (p *Proc) procType() ProcType {
	return p.procInternals.procType
}

// ------------------------------------------------------------------------------------------------
// CLIENTS PROC STRUCT
// ------------------------------------------------------------------------------------------------

// this is the internal view of a proc, ie what the client of the provider would create/run
type ProcInternals struct {
	sla        Tftick
	compDone   Tftick
	actualComp Tftick
	procType   ProcType
}

func (p *ProcInternals) String() string {
	return fmt.Sprintf("sla %v", p.sla)
}

func (p *ProcInternals) memUsed() Tmem {
	return p.procType.getMemoryUsage()
}

func newPrivProc(sla Tftick, procType ProcType) *ProcInternals {

	// get actual comp from a normal distribution, assuming the sla left a buffer
	slaWithoutBuffer := float64(sla) - procType.getExpectedSlaBuffer()*float64(sla)
	actualComp := Tftick(sampleNormal(slaWithoutBuffer, procType.getExpectedProcDeviationVariance(slaWithoutBuffer)))
	if actualComp < 0 {
		actualComp = Tftick(0.3)
	}

	return &ProcInternals{sla, 0, actualComp, procType}
}

func (p *ProcInternals) runTillOutOrDone(toRun Tftick) (Tftick, bool) {

	workLeft := p.actualComp - p.compDone

	if workLeft <= toRun {
		p.compDone = p.actualComp
		return workLeft, true
	} else {
		p.compDone += toRun
		// memUsage := rand.Int()%(PROC_MEM_CHANGE_MAX-PROC_MEM_CHANGE_MIN) + PROC_MEM_CHANGE_MIN
		// p.memUsed += Tmem(memUsage)
		// enforcing 0 <= memUsed <= MAX_MEM
		// p.memUsed = Tmem(math.Min(math.Max(float64(p.memUsed), 0), MAX_MEM))
		return toRun, false
	}
}
