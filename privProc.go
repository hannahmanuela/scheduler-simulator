package slasched

import "fmt"

// this is the internal view of a proc, ie what the client of the provider would create/run
type PrivProc struct {
	sla                    Tftick
	compDone               Tftick
	memUsed                Tmem
	actualComp             Tftick
	coresToComputeMultiple map[int]float64 // a measure of the parellelism of the task
}

func newPrivProc(sla Tftick, coresToComputeMultiple map[int]float64) *PrivProc {

	// get actual comp from a normal distribution, assuming the sla left a buffer
	slaWithoutBuffer := float64(sla) - PROC_SLA_EXPECTED_BUFFER*float64(sla)
	actualComp := Tftick(sampleNormal(slaWithoutBuffer, PROC_DEVIATION_FROM_SLA_VARIANCE))
	if actualComp < 0 {
		actualComp = Tftick(0.3)
	}

	return &PrivProc{sla, 0, 0, actualComp, coresToComputeMultiple}
}

// TODO: have this also increase mem usage
func (p *PrivProc) runTillOutOrDone(toRun computeTime) (Tftick, bool) {

	workLeft := p.actualComp - p.compDone
	productiveWork := p.getProductiveWork(toRun)

	if workLeft <= productiveWork {
		p.compDone = p.actualComp
		return p.computeTimeNeededToFinish(workLeft, toRun), true
	} else {
		p.compDone += productiveWork
		return productiveWork, false
	}
}

func (p *PrivProc) getProductiveWork(toRun computeTime) Tftick {
	fmt.Printf("running proc with %d cores, at an effectiveness multiple of %v\n", toRun.cores, p.coresToComputeMultiple[toRun.cores])
	return toRun.time * Tftick(p.coresToComputeMultiple[toRun.cores])
}

func (p *PrivProc) computeTimeNeededToFinish(workLeft Tftick, toRun computeTime) Tftick {
	return workLeft / Tftick(p.coresToComputeMultiple[toRun.cores])
}
