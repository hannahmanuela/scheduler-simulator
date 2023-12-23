package slasched

import (
	"math/rand"

	"gonum.org/v1/gonum/stat/distuv"
)

type SimpleWebsite struct {
	poisson *distuv.Poisson
}

func newSimpleWebsite(numMachines int) *SimpleWebsite {
	lambda := AVG_ARRIVAL_RATE_SMALL * (float64(numMachines))
	return &SimpleWebsite{poisson: &distuv.Poisson{Lambda: lambda}}
}

func (website *SimpleWebsite) genLoad(rand *rand.Rand, currTick Ttick) []*Proc {
	nproc := int(website.poisson.Rand())
	procs := make([]*Proc, nproc)
	for i := 0; i < nproc; i++ {
		procSLA := Tftick(0.95) // Ttick(uniform(rand))
		procs[i] = newProc(procSLA, currTick)
	}
	return procs
}
