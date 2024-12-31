package slasched

type HermodLB struct {
	currTickPtr     *Tftick
	nProcGenPerTick int
	machines        map[Tid]*HermodMachine
	GSSs            []*HermodGS
	roundRobinInd   int
}

func newHermodLB(numMachines int, numCores int, nGenPerTick int, nGSSs int, currTickPtr *Tftick) *HermodLB {

	mlb := &HermodLB{
		currTickPtr:   currTickPtr,
		machines:      map[Tid]*HermodMachine{},
		GSSs:          make([]*HermodGS, nGSSs),
		roundRobinInd: 0,
	}

	for i := 0; i < numMachines; i++ {
		mid := Tid(i)
		mlb.machines[Tid(i)] = newHermodMachine(mid, numCores, MEM_PER_MACHINE, mlb.currTickPtr, nGenPerTick)
	}

	numMachinesPerGS := int(numMachines / nGSSs)
	currBeg := Tid(0)
	for i := 0; i < nGSSs; i++ {
		currEnd := currBeg + Tid(numMachinesPerGS)
		if currEnd > Tid(numMachines)-Tid(numMachinesPerGS) {
			currEnd = Tid(numMachines)
		}
		machinesForGSS := make(map[Tid]*HermodMachine, 0)
		for id, m := range mlb.machines {
			if id >= currBeg && id < currEnd {
				machinesForGSS[id] = m
			}
		}
		mlb.GSSs[i] = newHermodGS(Tid(i), machinesForGSS, mlb.currTickPtr, mlb.nProcGenPerTick)
		currBeg = currEnd
	}

	return mlb
}

func (hlb *HermodLB) enqProc(proc *Proc) {

	// I think this is fine; in 4.3 they basically say it works

	hlb.GSSs[hlb.roundRobinInd].procQ = append(hlb.GSSs[hlb.roundRobinInd].procQ, proc)
	hlb.roundRobinInd += 1
	if hlb.roundRobinInd >= len(hlb.GSSs) {
		hlb.roundRobinInd = 0
	}

}

func (hlb *HermodLB) placeProcs() {

	for _, gs := range hlb.GSSs {
		gs.placeProcs()
	}

}

func (hlb *HermodLB) tick() {
	for _, m := range hlb.machines {
		m.tick()
	}
}
