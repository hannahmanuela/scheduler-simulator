package slasched

import (
	"math/rand"

	"gonum.org/v1/gonum/stat/distuv"
)

type SimpleWebsite struct {
	poisson *distuv.Poisson
}

func newSimpleWebsite(numMachines int) *SimpleWebsite {
	lambda := AVG_ARRIVAL_RATE * (float64(numMachines))
	return &SimpleWebsite{poisson: &distuv.Poisson{Lambda: lambda}}
}

func (website *SimpleWebsite) genLoad() []*ProcInternals {
	nproc := int(website.poisson.Rand())
	procs := make([]*ProcInternals, nproc)
	for i := 0; i < nproc; i++ {
		procSLA := Tftick(rand.Float64() * 5)
		procs[i] = newPrivProc(procSLA)
	}
	return procs
}
