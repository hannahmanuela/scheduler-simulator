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
	tenantId      Tid
	timeStarted   Tftick
	timeDone      Tftick
	priority      int
	computeTokens TCompTokens
	compDone      Tftick
	procInternals *ProcInternals
}

func (p *Proc) String() string {
	return strconv.Itoa(int(p.procId)) + ": " +
		"comp done: " + p.compDone.String() +
		", time started: " + p.timeStarted.String()
}

func newProvProc(procId Tid, currTick Tftick, privProc *ProcInternals) *Proc {
	return &Proc{
		procId:        procId,
		timeStarted:   currTick,
		timeDone:      0,
		computeTokens: privProc.compTokens,
		procInternals: privProc,
		tenantId:      privProc.tenantId,
	}
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
	compTokens TCompTokens
	actualComp Tftick
	procType   ProcType
	tenantId   Tid
}

func newPrivProc(compTokens TCompTokens, procType ProcType, tenantId Tid) *ProcInternals {

	actualComp := Tftick(sampleNormal(float64(procType.getExpectedComp()), procType.getExpectedProcDeviationVariance()))
	if actualComp < 0 {
		actualComp = Tftick(0.3)
	}

	return &ProcInternals{compTokens, actualComp, procType, tenantId}
}
