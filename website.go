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

func (website *SimpleWebsite) genLoad(rand *rand.Rand) []*PrivProc {
	nproc := int(website.poisson.Rand())
	procs := make([]*PrivProc, nproc)
	for i := 0; i < nproc; i++ {
		procSLA := Tftick(0.95) // Ttick(uniform(rand))
		procs[i] = newPrivProc(procSLA, genMapCoresToCompute(rand))
	}
	return procs
}

// generates a map from the number of cores to a multiple of compute,
// with a randomly selected slope and cutoff point after which adding more
// cores does the process no more good
func genMapCoresToCompute(rand *rand.Rand) map[int]float64 {
	slope := rand.Float64()
	cutoff := rand.Int()%7 + 1 // cutoff has to be at least 1
	toRet := make(map[int]float64, 0)
	for i := 1; i <= 8; i++ {
		if i < cutoff {
			toRet[i] = slope*(float64(i)-1) + 1
		} else {
			toRet[i] = slope*(float64(cutoff)-1) + 1
		}

	}
	return toRet
}
