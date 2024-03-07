package slasched

type Sched interface {
	getQ() *Queue
	tick()
	String() string
	memUsed() Tmem
	memUsage() float64
}
