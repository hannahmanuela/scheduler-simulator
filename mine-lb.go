package slasched

type MineLB struct {
	currTickPtr *Tftick

	machines      map[Tid]*Machine
	GSSs          []*GlobalSched
	roundRobinInd int
}

func newMineLB(numMachines int, numCores int, nGenPerTick int, nGSSs int, currTickPtr *Tftick) *MineLB {

	mlb := &MineLB{
		currTickPtr:   currTickPtr,
		machines:      map[Tid]*Machine{},
		GSSs:          make([]*GlobalSched, nGSSs),
		roundRobinInd: 0,
	}

	idleHeaps := make(map[Tid]*IdleHeap, nGSSs)
	for i := 0; i < nGSSs; i++ {
		idleHeap := &IdleHeap{
			heap: &MinHeap{},
		}
		idleHeaps[Tid(i)] = idleHeap
		mlb.GSSs[i] = newGolbalSched(i, mlb.machines, mlb.currTickPtr, nGenPerTick, idleHeap)
	}

	for i := 0; i < numMachines; i++ {
		mid := Tid(i)
		mlb.machines[Tid(i)] = newMachine(mid, idleHeaps, numCores, mlb.currTickPtr, nGenPerTick)
	}

	return mlb
}

func (mlb *MineLB) placeProc(provProc *Proc) {

	mlb.GSSs[mlb.roundRobinInd].multiq.enq(provProc)
	mlb.roundRobinInd += 1
	if mlb.roundRobinInd >= len(mlb.GSSs) {
		mlb.roundRobinInd = 0
	}

}

func (mlb *MineLB) placeProcs() {

	for _, gs := range mlb.GSSs {
		gs.placeProcs()
	}

}

func (mlb *MineLB) tick() {
	for _, m := range mlb.machines {
		m.tick()
	}
}
