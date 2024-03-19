package slasched

import (
	"math/rand"
)

// constants characterizing the wesbite traffic
const (
	// fraction of procs generated that are in each category
	FRACTION_PAGE_STATIC     = 0.6 // 0.5
	FRACTION_PAGE_DYNAMIC    = 0.35
	FRACTION_DATA_PROCESS_FG = 0.048 // 10
	FRACTION_DATA_PROCESS_BG = 0.002 // 05

	// Tick = 100 ms
	// the max/min value that a sla can have for the diff proc types - slas will have uniform random value in this range
	PAGE_STATIC_SLA_RANGE_MIN     = 0.001 // 0.1 ms
	PAGE_STATIC_SLA_RANGE_MAX     = 0.5
	PAGE_DYNAMIC_SLA_RANGE_MIN    = 0.001
	PAGE_DYNAMIC_SLA_RANGE_MAX    = 5
	DATA_PROCESS_FG_SLA_RANGE_MIN = 3
	DATA_PROCESS_FG_SLA_RANGE_MAX = 5
	DATA_PROCESS_BG_SLA_RANGE_MIN = 2
	DATA_PROCESS_BG_SLA_RANGE_MAX = 50

	// in MB
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

// variance of procs actual runtime to "expected" runtime (sla - sla * expected buffer)
func (pt ProcType) getExpectedProcDeviationVariance(slaWithoutBuffer float64) float64 {
	// page static, page dynamic, data process fg, data process bg
	return []float64{0.1, 0.1, 0.2, 0.5}[pt] * slaWithoutBuffer
}

// exected buffer between declared sla and average compute necessary, as a fraction of the sla
func (pt ProcType) getExpectedSlaBuffer() float64 {
	// page static, page dynamic, data process fg, data process bg
	return []float64{0.05, 0.05, 0.1, 0.3}[pt]
}

// the amount memory a proc of the given type will use (for now this is static)
func (pt ProcType) getMemoryUsage() Tmem {
	// page static, page dynamic, data process fg, data process bg
	return []Tmem{PAGE_STATIC_MEM_USG, PAGE_DYNAMIC_MEM_USG, DATA_PROCESS_FG_MEM_USG, DATA_PROCESS_BG_MEM_USG}[pt]
}

// type CacheClnt interface {
// 	put(k string, val string)
// 	get(k string) string
// }

type Website interface {
	genLoad(nProcs int) []*ProcInternals
}

// the website struct itself
type SimpleWebsite struct {
	// cacheClnt CacheClnt
}

func newSimpleWebsite() *SimpleWebsite {
	return &SimpleWebsite{}
}

// website function types:
// - respond to page requests (static, eg front page)
// - respond to page requests (dynamic, eg profile page)
// - process inputted user data (foreground, eg processes an uploading photo/video)
// - process user data (background, eg run data warehouse update flows)

func (website *SimpleWebsite) genLoad(nProcs int) []*ProcInternals {
	// nproc := int(website.poisson.Rand())
	procs := make([]*ProcInternals, 0)

	numStatic, numDynamic, numProcessFg, numProcessBg := website.genNumberOfProcs(nProcs)

	// gen all the proc types, for now this is manual
	procs = append(procs, website.genPageStaticProcs(numStatic)...)
	procs = append(procs, website.genPageDynamicProcs(numDynamic)...)
	procs = append(procs, website.genDataProcessFgProcs(numProcessFg)...)
	procs = append(procs, website.genDataProcessBgProcs(numProcessBg)...)

	return procs
}

func (website *SimpleWebsite) genNumberOfProcs(totalNumProcs int) (int, int, int, int) {

	numStatic := 0
	numDynamic := 0
	numProcessFg := 0
	numProcessBg := 0

	for i := 0; i < totalNumProcs; i++ {
		randVal := rand.Float64()
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

func (website *SimpleWebsite) genPageStaticProcs(numProcs int) []*ProcInternals {
	procs := make([]*ProcInternals, numProcs)
	for i := 0; i < numProcs; i++ {
		procSLA := Tftick(rand.Float64()*(PAGE_STATIC_SLA_RANGE_MAX-PAGE_DYNAMIC_SLA_RANGE_MIN)) + PAGE_STATIC_SLA_RANGE_MIN
		procs[i] = newPrivProc(procSLA, PAGE_STATIC)
		// fmt.Printf("created new static page proc: %v\n", procs[i])
	}
	return procs
}

func (website *SimpleWebsite) genPageDynamicProcs(numProcs int) []*ProcInternals {
	procs := make([]*ProcInternals, numProcs)
	for i := 0; i < numProcs; i++ {
		procSLA := Tftick(rand.Float64()*(PAGE_DYNAMIC_SLA_RANGE_MAX-PAGE_DYNAMIC_SLA_RANGE_MIN)) + PAGE_DYNAMIC_SLA_RANGE_MIN
		procs[i] = newPrivProc(procSLA, PAGE_DYNAMIC)
	}
	return procs
}

func (website *SimpleWebsite) genDataProcessFgProcs(numProcs int) []*ProcInternals {
	procs := make([]*ProcInternals, numProcs)
	for i := 0; i < numProcs; i++ {
		procSLA := Tftick(rand.Float64()*(DATA_PROCESS_FG_SLA_RANGE_MAX-DATA_PROCESS_FG_SLA_RANGE_MIN)) + DATA_PROCESS_FG_SLA_RANGE_MIN
		procs[i] = newPrivProc(procSLA, DATA_PROCESS_FG)
	}
	return procs
}

func (website *SimpleWebsite) genDataProcessBgProcs(numProcs int) []*ProcInternals {
	procs := make([]*ProcInternals, numProcs)
	for i := 0; i < numProcs; i++ {
		procSLA := Tftick(rand.Float64()*(DATA_PROCESS_BG_SLA_RANGE_MAX-DATA_PROCESS_BG_SLA_RANGE_MIN)) + DATA_PROCESS_BG_SLA_RANGE_MIN
		procs[i] = newPrivProc(procSLA, DATA_PROCESS_BG)
	}
	return procs
}

// func (website *SimpleWebsite) getFrontPage() string {
// 	return website.cacheClnt.get("home-page")
// }

// func (website *SimpleWebsite) getMyPage(userName string) string {
// 	user_posts_query := userName + "-posts"
// 	return website.cacheClnt.get(user_posts_query)
// }
