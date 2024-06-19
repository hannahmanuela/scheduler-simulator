package slasched

import (
	"fmt"
	"math"
	"math/rand"
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
	procTypeProfile  *ProvProcDistribution
}

func (p *Proc) String() string {
	return p.procInternals.String() +
		// ", deadline: " + p.timeShouldBeDone.String() +
		", ticks passed: " + p.ticksPassed.String()
	// ", procTypeProfile: " + p.procTypeProfile.String()
}

func newProvProc(currTick int, privProc *ProcInternals) *Proc {
	return &Proc{
		machineId:        -1,
		ticksPassed:      0,
		timeShouldBeDone: privProc.sla + Tftick(currTick),
		procInternals:    privProc,
	}
}

// runs proc for the number of ticks passed or until the proc is done,
// returning whether the proc is done and how many ticks were run, as well as whether the proc finished or was forcefully terminated for going over
func (p *Proc) runTillOutOrDone(toRun Tftick, currFtick Tftick) (Tftick, bool) {
	return p.procInternals.runTillOutOrDone(toRun, p.ticksPassed+currFtick)
}

func (p *Proc) effectiveSla() Tftick {
	return p.procInternals.sla
}

func (p *Proc) timeLeftOnSLA() Tftick {
	return p.effectiveSla() - p.ticksPassed
}

// based on profiling info
func (p *Proc) profilingExpectedCompLeft() Tftick {
	return Tftick(p.procTypeProfile.computeUsed.avg+p.procTypeProfile.computeUsed.stdDev) - (p.procInternals.compDone)
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
	sla             Tftick
	compDone        Tftick
	actualComp      Tftick
	ioNeeded        Tftick
	ioDone          Tftick
	nextUnblockedAt Tftick
	procType        ProcType
}

func (p *ProcInternals) String() string {
	return fmt.Sprintf("sla %v", p.sla)
}

func (p *ProcInternals) memUsed() Tmem {
	return p.procType.getMemoryUsage()
}

func newPrivProc(sla Tftick, ioNeeded Tftick, procType ProcType) *ProcInternals {

	// get actual comp from a normal distribution, assuming the sla left a buffer
	slaWithoutIo := sla - ioNeeded
	slaWithoutBuffer := float64(slaWithoutIo) - procType.getExpectedSlaBuffer()*float64(slaWithoutIo)
	actualComp := Tftick(sampleNormal(slaWithoutBuffer, procType.getExpectedProcDeviationVariance()))
	actualComp = min(sla-ioNeeded, actualComp)
	if actualComp < 0 {
		actualComp = Tftick(0.1)
	}

	return &ProcInternals{sla, 0, actualComp, ioNeeded, Tftick(0), Tftick(0), procType}
}

func (p *ProcInternals) runTillOutOrDone(toRun Tftick, currTick Tftick) (Tftick, bool) {

	workLeft := p.actualComp - p.compDone

	r := rand.Float64()
	r2 := rand.Float64()
	r3 := rand.Float64()

	if workLeft <= toRun {
		// if haven't used all the blocking, do so now
		if p.ioDone < p.ioNeeded {
			compUsed := Tftick(r2 * float64(workLeft))
			p.compDone += compUsed
			ioAdded := p.ioNeeded - p.ioDone
			// account for it now already
			p.ioDone += ioAdded

			p.nextUnblockedAt = currTick + compUsed + Tftick(ioAdded)

			return Tftick(r2 * float64(toRun)), false
		} else {
			p.compDone = p.actualComp
			return workLeft, true
		}
	} else {
		// randomly block (rn 30% of the time)
		if (p.ioDone < p.ioNeeded) && r < 0.3 {
			compUsed := Tftick(r2 * float64(toRun))
			p.compDone += compUsed
			ioAdded := math.Min(float64(p.ioNeeded-p.ioDone), r3*float64(p.ioNeeded))
			// account for it now already
			p.ioDone += Tftick(ioAdded)

			p.nextUnblockedAt = currTick + compUsed + Tftick(ioAdded)

			return Tftick(r2 * float64(toRun)), false
		}

		// we are not blocking, but also the ticks given is not enough to be done
		p.compDone += toRun
		return toRun, false
	}
}
