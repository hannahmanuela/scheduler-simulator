package slasched

import (
	"math"
)

type EDFProc struct {
	p  *Proc
	dl float32
}

type EDFLB struct {
	procs      []*EDFProc
	bigMachine *BigEDFMachine
}

func newEDFLB(numMachines int, numCores int, nGenPerTick int, currrTickPtr *Tftick) *EDFLB {
	ilb := &EDFLB{
		procs:      make([]*EDFProc, 0),
		bigMachine: newBigEDFMachine(numMachines*numCores, Tmem(numMachines*MEM_PER_MACHINE), currrTickPtr, nGenPerTick),
	}

	return ilb
}

func (elb *EDFLB) enqProc(proc *Proc) {

	topPrice := mapPriorityToDollars(N_PRIORITIES - 1)

	newDl := float32(proc.procInternals.actualComp) * (topPrice / proc.procInternals.willingToSpend)
	edfP := &EDFProc{p: proc, dl: newDl}

	elb.enq(edfP)

}

func (elb *EDFLB) placeProcs() {

	toReq := make([]*EDFProc, 0)

	p := elb.deq()

	for p != nil {
		placed := elb.bigMachine.potPlaceProc(p)

		if !placed {
			toReq = append(toReq, p)
		}
		p = elb.deq()
	}

	for _, p := range toReq {
		elb.enq(p)
	}

}

func (elb *EDFLB) tick() {
	elb.bigMachine.tick()
}

func (elb *EDFLB) deq() *EDFProc {
	minDl := float32(math.MaxFloat32)
	var procToRet *EDFProc
	idxToDel := -1

	for i, p := range elb.procs {
		if p.dl < minDl {
			minDl = p.dl
			procToRet = p
			idxToDel = i
		}
	}

	if idxToDel >= 0 {
		elb.procs = append(elb.procs[:idxToDel], elb.procs[idxToDel+1:]...)
	}

	return procToRet
}

func (elb *EDFLB) enq(newProc *EDFProc) {

	elb.procs = append(elb.procs, newProc)
}
