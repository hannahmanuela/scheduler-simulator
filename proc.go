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
	procId         Tid
	timeStarted    Tftick
	timePlaced     Tftick
	timeDone       Tftick
	compDone       Tftick
	memUsing       Tmem
	currentlyPaged bool
	totMemPaged    Tmem
	numTimesPaged  int
	procInternals  *ProcInternals
}

func (p *Proc) String() string {
	return strconv.Itoa(int(p.procId)) + ": " +
		", started: " + p.timeStarted.String() +
		", placed: " + p.timePlaced.String() +
		", done: " + p.timeDone.String() +
		", full comp " + p.procInternals.actualComp.String() +
		", comp done " + p.compDone.String() +
		", price: " + strconv.FormatFloat(float64(p.willingToSpend()), 'f', 3, 32) +
		", curr mem: " + strconv.Itoa(int(p.memUsing)) +
		", max mem: " + strconv.Itoa(int(p.procInternals.maxMem)) +
		", paged: " + strconv.FormatBool(p.currentlyPaged)
}

func newProvProc(procId Tid, currTick Tftick, privProc *ProcInternals) *Proc {
	return &Proc{
		procId:         procId,
		timeStarted:    currTick,
		timeDone:       0,
		compDone:       0,
		memUsing:       Tmem(privProc.initMem),
		currentlyPaged: false,
		totMemPaged:    0,
		numTimesPaged:  0,
		procInternals:  privProc,
	}
}

func (p *Proc) willingToSpend() float32 {
	return p.procInternals.willingToSpend
}

func (p *Proc) setCurrentlyPaged(newVal bool) {
	if newVal {
		p.totMemPaged += p.memUsing
		p.numTimesPaged += 1
	}
	p.currentlyPaged = newVal
}

func (p *Proc) runTillOutOrDone(toRun Tftick) (Tmem, Tftick, bool) {

	var memUseDelta Tmem
	if p.memUsing == p.procInternals.maxMem {
		memUseDelta = 0
	} else {
		memUseDelta = Tmem(r.Intn(int(p.procInternals.maxMem) - int(p.memUsing)))
		p.memUsing += memUseDelta
		if p.memUsing > p.procInternals.maxMem {
			fmt.Printf("WTF?? %v, %v", p.memUsing, p.procInternals.maxMem)
		}
	}

	workLeft := p.procInternals.actualComp - p.compDone

	if workLeft <= toRun {
		p.compDone = p.procInternals.actualComp
		return memUseDelta, workLeft, true
	} else {
		p.compDone += toRun
		return memUseDelta, toRun, false
	}
}

// ------------------------------------------------------------------------------------------------
// CLIENTS PROC STRUCT
// ------------------------------------------------------------------------------------------------

// this is the internal view of a proc, ie what the client of the provider would create/run
type ProcInternals struct {
	actualComp     Tftick
	willingToSpend float32
	initMem        Tmem
	maxMem         Tmem
}

func newPrivProc(actualComp float32, willingToSpend float32, initMem Tmem, maxMem Tmem) *ProcInternals {

	return &ProcInternals{Tftick(actualComp), willingToSpend, initMem, maxMem}
}
