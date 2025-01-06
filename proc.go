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
	timePlaced    Tftick
	timeDone      Tftick
	compDone      Tftick
	procInternals *ProcInternals
}

func (p *Proc) String() string {
	return strconv.Itoa(int(p.procId)) + ": " +
		", started: " + p.timeStarted.String() +
		", placed: " + p.timePlaced.String() +
		", done: " + p.timeDone.String() +
		", full comp " + p.procInternals.actualComp.String() +
		", comp done " + p.compDone.String() +
		", price: " + strconv.FormatFloat(float64(p.willingToSpend()), 'f', 3, 32) +
		", mem: " + strconv.Itoa(int(p.maxMem()))
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
	compGuess      Tftick
	willingToSpend float32
	maxMem         Tmem
}

func newPrivProc(actualComp float32, compGuess float32, willingToSpend float32, maxMem int) *ProcInternals {

	return &ProcInternals{Tftick(actualComp), Tftick(compGuess), willingToSpend, Tmem(maxMem)}
}
