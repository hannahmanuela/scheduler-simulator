package slasched

import "math"

// constants characterizing the wesbite traffic
const (
	N_PRIORITIES = 5

	MIN_COMP     = 0.2
	AVG_COMP     = 2
	STD_DEV_COMP = 5
	MAX_COMP     = 100

	INIT_MEM = 100
	MAX_MEM  = 10000

	PARETO_ALPHA = 25
)

type LoadGen interface {
	genLoad(nProcs int) []*ProcInternals
}

// the website struct itself
type LoadGenT struct {
}

func newLoadGen() *LoadGenT {
	return &LoadGenT{}
}

func (lg *LoadGenT) genLoad(nProcs int) []*ProcInternals {
	procs := make([]*ProcInternals, nProcs)

	for i := 0; i < nProcs; i++ {

		minComp := math.Max(math.Min(sampleNormal(AVG_COMP, STD_DEV_COMP), MAX_COMP), MIN_COMP)
		actualComp := ParetoSample(PARETO_ALPHA, float64(minComp))

		priority := genRandPriority()
		willingToSpend := mapPriorityToDollars(priority)

		// maxMem := Tmem(ParetoSample(PARETO_ALPHA, (MAX_COMP - INIT_MEM)))
		maxMem := Tmem(INIT_MEM + r.Intn(MAX_MEM-INIT_MEM))

		procs[i] = newPrivProc(float32(actualComp), willingToSpend, Tmem(INIT_MEM), maxMem)
	}

	return procs
}
