package slasched

import (
	"fmt"
)

// this is the external view of a clients proc, that includes provider-created/maintained metadata, etc
type ProvProc struct {
	ticksPassed      Tftick
	timeShouldBeDone Tftick
	// numCoresHelpful  int // initialized as 1, added to as we give more cores, until they don't help anymore
	privProc *PrivProc
}

func (p *ProvProc) String() string {
	return fmt.Sprintf("{sla %v actualComp %v compDone %v memUsed %d}", p.privProc.sla, p.privProc.actualComp, p.privProc.compDone, p.privProc.memUsed)
}

func newProvProc(currTick Ttick, privProc *PrivProc) *ProvProc {
	return &ProvProc{0, privProc.sla + Tftick(currTick), privProc}
}

// runs proc for the number of ticks passed or until the proc is done,
// returning whether the proc is done and how many ticks were run.
func (p *ProvProc) runTillOutOrDone(toRun computeTime) (Tftick, bool) {
	ticksUsed, done := p.privProc.runTillOutOrDone(toRun)
	// TODO: update numCoresHelpful based on how much work is done?
	// is that something that schedulers know?
	return ticksUsed, done
}

func (p *ProvProc) timeLeftOnSLA() Tftick {
	return p.privProc.sla - p.ticksPassed
}

func (p *ProvProc) memUsed() Tmem {
	return p.privProc.memUsed
}
