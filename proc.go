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
	machineId     Tid
	timeStarted   Tftick
	timeDone      Tftick
	deadline      Tftick
	maxComp       Tftick
	compDone      Tftick
	procInternals *ProcInternals
}

func (p *Proc) String() string {
	return strconv.Itoa(int(p.procId)) + ": " +
		"comp done: " + p.compDone.String() +
		", deadline: " + p.deadline.String() +
		", time started: " + p.timeStarted.String()
}

func newProvProc(procId Tid, currTick Tftick, privProc *ProcInternals) *Proc {
	return &Proc{
		procId:        procId,
		machineId:     -1,
		timeStarted:   currTick,
		timeDone:      0,
		deadline:      privProc.deadline,
		maxComp:       privProc.maxComp,
		procInternals: privProc,
	}
}

// returns the deadline (relative, offset by time started)
func (p *Proc) getRelDeadline() Tftick {
	return p.timeStarted + p.deadline
}

func (p *Proc) getSlack(currTime Tftick) Tftick {
	ogSlack := p.deadline - p.maxComp
	return ogSlack - p.waitTime(currTime)
}

func (p *Proc) waitTime(currTime Tftick) Tftick {
	return (currTime - p.timeStarted) - p.compUsed()
}

func (p *Proc) getExpectedCompLeft() Tftick {
	return p.maxComp - p.compUsed()
}

func (p *Proc) compUsed() Tftick {
	return p.compDone
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
	deadline   Tftick
	maxComp    Tftick
	actualComp Tftick
	procType   ProcType
}

func (p *ProcInternals) memUsed() Tmem {
	return p.procType.getMemoryUsage()
}

func newPrivProc(sla Tftick, maxComp Tftick, procType ProcType) *ProcInternals {

	// get actual comp from a normal distribution, assuming the sla left a buffer
	slaWithoutBuffer := float64(sla) - procType.getExpectedSlaBuffer()*float64(sla)
	actualComp := Tftick(sampleNormal(slaWithoutBuffer, procType.getExpectedProcDeviationVariance()))
	if actualComp < 0 {
		actualComp = Tftick(0.3)
	} else if actualComp > maxComp {
		actualComp = maxComp
	}

	return &ProcInternals{sla, maxComp, actualComp, procType}
}
