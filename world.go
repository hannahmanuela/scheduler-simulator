package slasched

const (
	MAX_MEM_PER_MACHINE = 32000 // the amount of memory every core will have, in MB

	IDLE_HEAP_THRESHOLD = 5

	TICKS_WAIT_LOAD_CHANGES = 100
	INITIAL_LOAD            = 4
	THRESHOLD_RATIO_MIN     = 0.1 // min max ratio below which we add load
	THRESHOLD_RATIO_MAX     = 0.5 // min max ratio above which we reduce load

	VERBOSE_LB_STATS            = true
	VERBOSE_SCHED_STATS         = true
	VERBOSE_WORLD_STATS         = true
	VERBOSE_MACHINE_USAGE_STATS = true
	VERBOSE_PRESSURE_VALS       = true
)

type World struct {
	currTick        Tftick
	numProcsToGen   int
	currProcNum     int
	lastChangedLoad int
	machines        map[Tid]*Machine
	lb              *GlobalSched
	app             Website
}

func newWorld(numMachines int, numCores int) *World {
	w := &World{
		currTick:        Tftick(0),
		machines:        map[Tid]*Machine{},
		numProcsToGen:   INITIAL_LOAD,
		lastChangedLoad: 0,
	}
	idleHeap := &IdleHeap{
		heap: &MinHeap{},
	}
	for i := 0; i < numMachines; i++ {
		mid := Tid(i)
		w.machines[Tid(i)] = newMachine(mid, idleHeap, numCores, &w.currTick)
	}
	w.lb = newLoadBalancer(w.machines, &w.currTick, idleHeap)
	return w
}

func (w *World) String() string {
	str := "machines: \n"
	for _, m := range w.machines {
		str += "   " + m.String()
	}
	return str
}

func (w *World) genLoad(nProcs int) int {
	userProcs := w.app.genLoad(nProcs)
	sumTicksAdded := Tftick(0)
	for _, up := range userProcs {
		sumTicksAdded += up.actualComp
		provProc := newProvProc(Tid(w.currProcNum), w.currTick, up)
		w.currProcNum += 1
		w.lb.putProc(provProc)
	}
	return len(userProcs)
}

func (w *World) compute() {
	for _, m := range w.machines {
		m.sched.tick()
	}
}

func (w *World) printAllProcs() {
	for _, m := range w.machines {
		m.sched.printAllProcs()
	}
}

func (w *World) Tick(numProcs int) {
	if VERBOSE_LB_STATS {
		w.printAllProcs()
	}
	// enqueues things into the procq
	w.genLoad(numProcs)
	// dequeues things from procq to machines
	w.lb.placeProcs()
	// runs each machine for a tick
	w.compute()
	w.currTick += 1
}

func (w *World) Run(nTick int, nProcsPerTick int) {
	for i := 0; i < nTick; i++ {
		w.Tick(nProcsPerTick)
	}
}
