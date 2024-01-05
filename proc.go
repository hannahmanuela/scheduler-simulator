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
	ticksPassed      Tftick
	timeShouldBeDone Tftick
	procInternals    *ProcInternals
}

func (p *Proc) String() string {
	return fmt.Sprintf("{sla %v actualComp %v compDone %v memUsed %d}", p.procInternals.sla,
		p.procInternals.actualComp, p.procInternals.compDone, p.procInternals.memUsed)
}

func newProvProc(currTick Ttick, privProc *ProcInternals) *Proc {
	return &Proc{0, privProc.sla + Tftick(currTick), privProc}
}

// runs proc for the number of ticks passed or until the proc is done,
// returning whether the proc is done and how many ticks were run.
func (p *Proc) runTillOutOrDone(toRun Tftick) (Tftick, bool) {
	ticksUsed, done := p.procInternals.runTillOutOrDone(toRun)
	return ticksUsed, done
}

// difference between the time that has passed since the proc started and its SLA
func (p *Proc) timeLeftOnSLA() Tftick {
	return p.procInternals.sla - p.ticksPassed
}

func (p *Proc) memUsed() Tmem {
	return p.procInternals.memUsed
}

// returns a measure of how killable a proc is
// (based on how much memory its using, how long it has already been running, and what its sla is)
func (p *Proc) killableScore() float64 {
	// higher score: memUsed, sla
	// lower score: compDone
	return (float64(p.memUsed()) + float64(p.procInternals.sla)) - float64(p.procInternals.compDone)
}

// ------------------------------------------------------------------------------------------------
// CLIENTS PROC STRUCT
// ------------------------------------------------------------------------------------------------

// this is the internal view of a proc, ie what the client of the provider would create/run
type ProcInternals struct {
	sla        Tftick
	compDone   Tftick
	memUsed    Tmem
	actualComp Tftick
}

func newPrivProc(sla Tftick) *ProcInternals {

	// get actual comp from a normal distribution, assuming the sla left a buffer
	slaWithoutBuffer := float64(sla) - PROC_SLA_EXPECTED_BUFFER*float64(sla)
	actualComp := Tftick(sampleNormal(slaWithoutBuffer, PROC_DEVIATION_FROM_SLA_VARIANCE))
	if actualComp < 0 {
		actualComp = Tftick(0.3)
	}

	return &ProcInternals{sla, 0, 0, actualComp}
}

func (p *ProcInternals) runTillOutOrDone(toRun Tftick) (Tftick, bool) {

	workLeft := p.actualComp - p.compDone

	if workLeft <= toRun {
		p.compDone = p.actualComp
		fmt.Printf("proc done: had %v work left\n", workLeft)
		return workLeft, true
	} else {
		p.compDone += toRun
		memUsage := rand.Int()%10 - 3 // between -3 and +8
		p.memUsed += Tmem(memUsage)
		p.memUsed = Tmem(math.Max(float64(p.memUsed), 0))
		fmt.Printf("adding %v memory, for a total of %v\n", memUsage, p.memUsed)
		return toRun, false
	}
}
