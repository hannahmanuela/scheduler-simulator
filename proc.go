package slasched

import (
	"strconv"
)

// ------------------------------------------------------------------------------------------------
// PROVIDER PROC STRUCT
// ------------------------------------------------------------------------------------------------

// this is the external view of a clients proc, that includes provider-created/maintained metadata, etc
type Proc struct {
	procId        Tid
	timeStarted   Tftick
	timeDone      Tftick
	compDone      Tftick
	procInternals *ProcInternals
}

func (p *Proc) String() string {
	return strconv.Itoa(int(p.procId)) + ": " +
		", time started: " + p.timeStarted.String()
}

func newProvProc(procId Tid, currTick Tftick, privProc *ProcInternals) *Proc {
	return &Proc{
		procId:        procId,
		timeStarted:   currTick,
		timeDone:      0,
		compDone:      0,
		procInternals: privProc,
	}
}

func (p *Proc) willingToSpend() float32 {
	return p.procInternals.willingToSpend
}

func (p *Proc) maxMem() Tmem {
	return p.procInternals.maxMem
}

func (p *Proc) runTillOutOrDone(toRun Tftick) (Tftick, bool) {

	workLeft := p.procInternals.actualComp - p.compDone

	if workLeft <= toRun {
		p.compDone = p.procInternals.actualComp
		return workLeft, true
	} else {
		p.compDone += toRun
		return toRun, false
	}
}

// ------------------------------------------------------------------------------------------------
// CLIENTS PROC STRUCT
// ------------------------------------------------------------------------------------------------

// this is the internal view of a proc, ie what the client of the provider would create/run
type ProcInternals struct {
	actualComp     Tftick
	willingToSpend float32
	maxMem         Tmem
}

func newPrivProc(expectedComp float32, compVar float32, willingToSpend float32, maxMem Tmem) *ProcInternals {

	actualComp := Tftick(sampleNormal(float64(expectedComp), float64(compVar)))
	if actualComp < 0 {
		actualComp = Tftick(0.3)
	}

	return &ProcInternals{actualComp, willingToSpend, maxMem}
}
