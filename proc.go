package slasched

import (
	"fmt"
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
	migrated         bool
}

func (p *Proc) String() string {
	return p.procInternals.String() + ", deadline: " + p.timeShouldBeDone.String()
}

func newProvProc(currTick Ttick, privProc *ProcInternals) *Proc {
	return &Proc{
		machineId:        -1,
		ticksPassed:      0,
		timeShouldBeDone: privProc.sla + Tftick(currTick),
		procInternals:    privProc,
		migrated:         false}
}

// runs proc for the number of ticks passed or until the proc is done,
// returning whether the proc is done and how many ticks were run, as well as whether the proc finished or was forcefully terminated for going over
func (p *Proc) runTillOutOrDone(toRun Tftick) (Tftick, bool) {
	return p.procInternals.runTillOutOrDone(toRun)
}

func (p *Proc) getSla() Tftick {
	return p.procInternals.sla
}

func (p *Proc) memUsed() Tmem {
	return p.procInternals.memUsed()
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
	return fmt.Sprintf("{type %v, sla %v actualComp %v compDone %v memUsed %d}", p.procType, p.sla,
		p.actualComp, p.compDone, p.memUsed())
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
		if VERBOSE_PROC {
			fmt.Printf("proc done: had %v work left\n", workLeft)
		}
		return workLeft, true
	} else {
		p.compDone += toRun
		// memUsage := rand.Int()%(PROC_MEM_CHANGE_MAX-PROC_MEM_CHANGE_MIN) + PROC_MEM_CHANGE_MIN
		// p.memUsed += Tmem(memUsage)
		// enforcing 0 <= memUsed <= MAX_MEM
		// p.memUsed = Tmem(math.Min(math.Max(float64(p.memUsed), 0), MAX_MEM))
		// fmt.Printf("adding %v memory, for a total of %v\n", memUsage, p.memUsed)
		return toRun, false
	}
}
