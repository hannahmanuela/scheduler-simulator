package slasched

import (
	"fmt"
	"math"
)

type HermodGS struct {
	gsId            Tid
	machines        map[Tid]*HermodMachine
	procQ           []*Proc
	currTickPtr     *Tftick
	nProcGenPerTick int
}

func newHermodGS(id Tid, machines map[Tid]*HermodMachine, currTickPtr *Tftick, nProcGenPerTick int) *HermodGS {

	hgs := &HermodGS{
		gsId:            id,
		machines:        machines,
		procQ:           make([]*Proc, 0),
		currTickPtr:     currTickPtr,
		nProcGenPerTick: nProcGenPerTick,
	}
	return hgs
}

func (hgs *HermodGS) placeProcs() {

	logWrite(HERMOD_SCHED, "\n")

	toReq := make([]*Proc, 0)

	for _, p := range hgs.procQ {
		// place given proc

		machineToUse := hgs.pickMachine(p)

		toWrite := fmt.Sprintf("%v, GS %v placing proc %v \n", int(*hgs.currTickPtr), hgs.gsId, p.procId)
		logWrite(HERMOD_SCHED, toWrite)

		if machineToUse == nil {
			logWrite(HERMOD_SCHED, "    -> nothing avail \n")
			toReq = append(toReq, p)
			continue
		}

		machineToUse.placeProc(p)
		toWrite = fmt.Sprintf("    -> chose %v \n", machineToUse.machineId)
		logWrite(HERMOD_SCHED, toWrite)

	}

	hgs.procQ = toReq
}

func (hgs *HermodGS) pickMachine(procToPlace *Proc) *HermodMachine {

	// power of k choices - do hermod hybrid load balancing thing among the sampled machines

	// right now always doing high load scenario stuff

	var machineToUse *HermodMachine
	machinesToTry := pickRandomElements(Values(hgs.machines), K_CHOICES_DOWN)
	leastNumProcs := math.MaxInt

	for _, m := range machinesToTry {
		if len(m.procs) < leastNumProcs {
			leastNumProcs = len(m.procs)
			machineToUse = m
		}
	}

	return machineToUse

}
