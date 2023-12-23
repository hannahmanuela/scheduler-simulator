package slasched

import (
	"fmt"
	"math/rand"
)

type Proc struct {
	ticksPassed      Tftick
	sla              Tftick
	compDone         Tftick
	memUsed          Tmem
	timeShouldBeDone Tftick
	actualComp       Tftick // DON'T USE THIS OUTSIDE PROC
}

func (p *Proc) String() string {
	return fmt.Sprintf("{sla %v actualComp %v compDone %v memUsed %d}", p.sla, p.actualComp, p.compDone, p.memUsed)
}

func newProc(sla Tftick, currTick Ttick) *Proc {
	slaWithoutBuffer := float64(sla) - PROC_SLA_EXPECTED_BUFFER*float64(sla)
	actualComp := Tftick(sampleNormal(slaWithoutBuffer, PROC_DEVIATION_FROM_SLA_VARIANCE))
	return &Proc{0, sla, 0, 0, sla + Tftick(currTick), actualComp}
}

// runs proc for the number of ticks passed or until the proc is done,
// returning whether the proc is done and how many ticks were run.
// sets the compDone value
func (p *Proc) runTillOutOrDone(ticksToRun Tftick) (Tftick, bool) {
	workLeft := p.actualComp - p.compDone
	if workLeft <= ticksToRun {
		p.compDone = p.actualComp
		return workLeft, true
	} else {
		p.compDone += ticksToRun
		return ticksToRun, false
	}
}

func (p *Proc) timeLeftOnSLA() Tftick {
	return p.sla - p.ticksPassed
}

func sampleNormal(mu, sigma float64) float64 {
	return rand.NormFloat64()*float64(sigma) + float64(mu)
}
