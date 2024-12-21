package slasched

import "math"

// constants characterizing the wesbite traffic
const (
	N_PRIORITIES = 5

	MIN_COMP     = 0.2
	AVG_COMP     = 2
	STD_DEV_COMP = 5
	MAX_COMP     = 100

	MIN_MEM = 1
	MAX_MEM = 10000

	PARETO_ALPHA = 25
)

type Website interface {
	genLoad(nProcs int, tenantId Tid) []*ProcInternals
}

// the website struct itself
type SimpleWebsite struct {
}

func newSimpleWebsite() *SimpleWebsite {
	return &SimpleWebsite{}
}

// 0 is the cheapest, 5 the most expensive
func mapPriorityToDollars(priority int) float32 {
	return []float32{0.3, 0.7, 1.0, 1.5, 2}[priority]
}

func (website *SimpleWebsite) genLoad(nProcs int, tenantId Tid) []*ProcInternals {
	// nproc := int(website.poisson.Rand())
	procs := make([]*ProcInternals, nProcs)

	for i := 0; i < nProcs; i++ {

		minComp := math.Max(math.Min(sampleNormal(AVG_COMP, STD_DEV_COMP), MAX_COMP), MIN_COMP)
		actualComp := ParetoSample(PARETO_ALPHA, float64(minComp))

		priority := r.Intn(N_PRIORITIES)
		willingToSpend := mapPriorityToDollars(priority)

		maxMem := MIN_MEM + r.Intn(MAX_MEM-MIN_MEM)

		procs[i] = newPrivProc(float32(actualComp), willingToSpend, maxMem, tenantId)
	}

	return procs
}
