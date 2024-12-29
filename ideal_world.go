package slasched

type IdealWorld struct {
	currTick      Tftick
	numProcsToGen int
	currProcNum   int
	idealDC       *IdealDC
	idealMultiQ   MultiQueue
	loadGen       LoadGen
}

func newIdealWorld(numMachines int, numCores int, nGenPerTick int) *IdealWorld {

	w := &IdealWorld{
		currTick:      Tftick(0),
		numProcsToGen: nGenPerTick,
		idealMultiQ:   NewMultiQ(),
		loadGen:       newLoadGen(),
	}

	w.idealDC = newIdealDC(numMachines*numCores, Tmem(numMachines*MEM_PER_MACHINE), &w.currTick, nGenPerTick)

	return w
}

func (iw *IdealWorld) genLoad(nProcs int) int {

	userProcs := iw.loadGen.genLoad(nProcs)

	for _, up := range userProcs {
		provProc := newProvProc(Tid(iw.currProcNum), iw.currTick, up)
		iw.idealMultiQ.enq(provProc)
		iw.currProcNum += 1
	}
	return len(userProcs)
}

func (iw *IdealWorld) placeProcs() {

	toReq := make([]*Proc, 0)

	p := iw.idealMultiQ.deq(iw.currTick)

	for p != nil {
		placed := iw.idealDC.potPlaceProc(p)

		if !placed {
			toReq = append(toReq, p)
		}
		p = iw.idealMultiQ.deq(iw.currTick)
	}

	for _, p := range toReq {
		iw.idealMultiQ.enq(p)
	}

}

func (iw *IdealWorld) Tick(numProcs int) {
	iw.genLoad(numProcs)

	iw.placeProcs()

	iw.idealDC.tick()

	iw.currTick += 1
}

func (iw *IdealWorld) Run(nTick int) {
	for i := 0; i < nTick; i++ {
		iw.Tick(iw.numProcsToGen)
	}
}
