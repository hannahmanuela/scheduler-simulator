package slasched

// constants characterizing the wesbite traffic
const (
	// fraction of procs generated that are in each category
	FRACTION_PAGE_STATIC     = 0.6 // 0.5
	FRACTION_PAGE_DYNAMIC    = 0.35
	FRACTION_DATA_PROCESS_FG = 0.047 // 10
	FRACTION_DATA_PROCESS_BG = 0.003 // 05

	// mem usage, in MB
	PAGE_STATIC_MEM_USG     = 20
	PAGE_DYNAMIC_MEM_USG    = 300
	DATA_PROCESS_FG_MEM_USG = 1000
	DATA_PROCESS_BG_MEM_USG = 10000
)

// the types of procs the website will have
type ProcType int

const (
	PAGE_STATIC ProcType = iota
	PAGE_DYNAMIC
	DATA_PROCESS_FG
	DATA_PROCESS_BG
)

func (pt ProcType) String() string {
	return []string{"page static", "page dynamic", "data process fg", "data process bg"}[pt]
}

// returns std dev (sigma)
// this is used as the sigma in getting the actual runtime
func (pt ProcType) getExpectedProcDeviationVariance() float64 {
	// page static: 0.5ms, page dynamic: 2ms, data process fg: 10ms, data process bg: 500ms
	return []float64{0.1, 0.4, 2, 100}[pt]
}

func (pt ProcType) getExpectedComp() int {
	// page static: 5ms, page dynamic: 20ms, data process fg: 500ms, data process bg: 5ms
	return []int{1, 4, 100, 1000}[pt]
}

// // the amount memory a proc of the given type will use (for now this is static)
func (pt ProcType) getMemoryUsage() Tmem {
	// page static, page dynamic, data process fg, data process bg
	return []Tmem{PAGE_STATIC_MEM_USG, PAGE_DYNAMIC_MEM_USG, DATA_PROCESS_FG_MEM_USG, DATA_PROCESS_BG_MEM_USG}[pt]
}

func (pt ProcType) getWillingToSpend() float32 {
	// page static, page dynamic, data process fg, data process bg
	return []float32{3, 3.5, 2, 1.5}[pt]
}

type Website interface {
	genLoad(nProcs int, tenantId Tid) []*ProcInternals
}

// the website struct itself
type SimpleWebsite struct {
}

func newSimpleWebsite() *SimpleWebsite {
	return &SimpleWebsite{}
}

// website function types:
// - respond to page requests (static, eg front page)
// - respond to page requests (dynamic, eg profile page)
// - process inputted user data (foreground, eg processes an uploading photo/video)
// - process user data (background, eg run data warehouse update flows)

func (website *SimpleWebsite) genLoad(nProcs int, tenantId Tid) []*ProcInternals {
	// nproc := int(website.poisson.Rand())
	procs := make([]*ProcInternals, 0)

	numStatic, numDynamic, numProcessFg, numProcessBg := website.genNumberOfProcs(nProcs)

	// gen all the proc types, for now this is manual
	procs = append(procs, website.genProcsOfType(PAGE_STATIC, numStatic, tenantId)...)
	procs = append(procs, website.genProcsOfType(PAGE_DYNAMIC, numDynamic, tenantId)...)
	procs = append(procs, website.genProcsOfType(DATA_PROCESS_FG, numProcessFg, tenantId)...)
	procs = append(procs, website.genProcsOfType(DATA_PROCESS_BG, numProcessBg, tenantId)...)

	return procs
}

func (website *SimpleWebsite) genNumberOfProcs(totalNumProcs int) (int, int, int, int) {

	numStatic := 0
	numDynamic := 0
	numProcessFg := 0
	numProcessBg := 0

	for i := 0; i < totalNumProcs; i++ {
		randVal := r.Float64()
		if randVal < FRACTION_DATA_PROCESS_BG {
			numProcessBg += 1
		} else if randVal < FRACTION_DATA_PROCESS_BG+FRACTION_DATA_PROCESS_FG {
			numProcessFg += 1
		} else if randVal < FRACTION_PAGE_DYNAMIC+FRACTION_DATA_PROCESS_FG+FRACTION_DATA_PROCESS_BG {
			numDynamic += 1
		} else {
			numStatic += 1
		}
	}

	return numStatic, numDynamic, numProcessFg, numProcessBg

}

func (website *SimpleWebsite) genProcsOfType(typeWanted ProcType, numProcs int, tenantId Tid) []*ProcInternals {
	procs := make([]*ProcInternals, numProcs)
	for i := 0; i < numProcs; i++ {
		procs[i] = newPrivProc(float32(typeWanted.getExpectedComp()), float32(typeWanted.getExpectedProcDeviationVariance()), typeWanted.getWillingToSpend(), typeWanted.getMemoryUsage(), tenantId)
	}
	return procs
}
