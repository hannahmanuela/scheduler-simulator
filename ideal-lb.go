package slasched

type IdealLB struct {
	currTickPtr *Tftick

	multiQ     MultiQueue
	bigMachine *BigIdealMachine
}

func newIdealLB(numMachines int, numCores int, nGenPerTick int, currrTickPtr *Tftick) *IdealLB {
	ilb := &IdealLB{
		currTickPtr: currrTickPtr,
		multiQ:      NewMultiQ(),
		bigMachine:  newBigIdealMachine(numMachines*numCores, Tmem(numMachines*MEM_PER_MACHINE), currrTickPtr, nGenPerTick),
	}

	return ilb
}

func (ilb *IdealLB) placeProc(proc *Proc) {

	ilb.multiQ.enq(proc)

}

func (ilb *IdealLB) placeProcs() {

	toReq := make([]*Proc, 0)

	p := ilb.multiQ.deq(*ilb.currTickPtr)

	for p != nil {
		placed := ilb.bigMachine.potPlaceProc(p)

		if !placed {
			toReq = append(toReq, p)
		}
		p = ilb.multiQ.deq(*ilb.currTickPtr)
	}

	for _, p := range toReq {
		ilb.multiQ.enq(p)
	}

}

func (ilb *IdealLB) tick() {
	ilb.bigMachine.tick()
}
