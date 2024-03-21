package slasched

import (
	"fmt"
	"math"
)

type Distribution struct {
	avg    float64
	count  int
	stdDev float64
}

func (d *Distribution) update(newVal float64) {
	d.avg = (d.avg*float64(d.count) + float64(newVal)) / float64(d.count+1)
	d.stdDev = math.Sqrt((math.Pow(d.stdDev, 2)*float64(d.count) + math.Pow(newVal-d.avg, 2)) / float64(d.count+1))
	d.count += 1
}

type ProvProcDistribution struct {
	memUsg      Distribution
	computeUsed Distribution
}

func newProcProcDistribution(initMem Tmem, initCompute Tftick) *ProvProcDistribution {
	return &ProvProcDistribution{
		memUsg:      Distribution{float64(initMem), 1, 0},
		computeUsed: Distribution{float64(initCompute), 1, 0},
	}
}

func (ppt *ProvProcDistribution) String() string {
	return fmt.Sprintf("avgComp: %v, stdDev: %v; avg memL %v, stdDev: %v\n", ppt.computeUsed.avg, ppt.computeUsed.stdDev, ppt.memUsg.avg, ppt.memUsg.stdDev)
}

func (ppt *ProvProcDistribution) updateMem(val Tmem) {
	ppt.memUsg.update(float64(val))
}

func (ppt *ProvProcDistribution) updateCompute(val Tftick) {
	ppt.computeUsed.update(float64(val))
}
