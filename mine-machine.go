package slasched

import (
	"fmt"
	"math"
)

const (
	TICK_SCHED_THRESHOLD = 0.001 // amount of ticks after which I stop scheduling; given that 1 tick = 5ms (see website.go)
)

type Tid int

type Ttickmap map[Tid]Tftick
type Tprocmap map[Tid]int

type Machine struct {
	machineId               Tid
	numCores                int
	activeQ                 *Queue
	idleHeaps               map[Tid]*IdleHeap
	currHeapGSS             Tid
	currTickPtr             *Tftick
	worldNumProcsGenPerTick int
}

func newMachine(mid Tid, idleHeaps map[Tid]*IdleHeap, numCores int, currTickPtr *Tftick, nGenPerTick int) *Machine {

	sd := &Machine{
		machineId:               mid,
		numCores:                numCores,
		activeQ:                 newQueue(),
		idleHeaps:               idleHeaps,
		currHeapGSS:             -1,
		currTickPtr:             currTickPtr,
		worldNumProcsGenPerTick: nGenPerTick,
	}

	// add machine to an idle heap
	heapsToLookAt := pickRandomElementsMap(sd.idleHeaps, K_CHOICES_UP)

	var gsHeapToUse Tid
	minLength := math.MaxInt

	for gsId, possHeap := range heapsToLookAt {
		possHeap.lock.Lock()
		if possHeap.heap.Len() < minLength {
			minLength = possHeap.heap.Len()
			gsHeapToUse = gsId
		}
		possHeap.lock.Unlock()
	}

	sd.currHeapGSS = gsHeapToUse
	heapToUse := sd.idleHeaps[gsHeapToUse]
	heapToUse.lock.Lock()
	toPush := TIdleMachine{
		memAvail:           MEM_PER_MACHINE,
		highestCostRunning: -1,
		qlen:               0,
		machine:            Tid(sd.machineId),
	}
	heapToUse.heap.Push(toPush)
	heapToUse.lock.Unlock()

	return sd
}

func (sd *Machine) String() string {
	return fmt.Sprintf("machine scheduler: %v", sd.machineId)
}

func (sd *Machine) tick() {
	sd.simulateRunProcs()
}

func (sd *Machine) memFree() Tmem {

	memUsed := Tmem(0)

	for _, p := range sd.activeQ.getQ() {
		memUsed += p.maxMem()
	}
	return MEM_PER_MACHINE - memUsed
}

func (sd *Machine) okToPlace(newProc *Proc) float32 {

	// if it just fits in terms of memory do it
	if newProc.maxMem() < sd.memFree() {
		return 0
	}

	// if it doesn't fit, look if there a good proc to kill? (/a combination of procs? can add that later)
	_, minTimeToProfit := sd.activeQ.checkKill(newProc)

	return minTimeToProfit
}

func (sd *Machine) placeProc(newProc *Proc, fromGs Tid) (bool, TIdleMachine) {

	newProc.timePlaced = *sd.currTickPtr

	ogMemFree := sd.memFree()

	if newProc.maxMem() < sd.memFree() {
		sd.activeQ.enq(newProc)

	} else {
		// if it doesn't fit, look if there a good proc to kill? (/a combination of procs? can add that later)
		procToKill, _ := sd.activeQ.checkKill(newProc)

		sd.activeQ.kill(procToKill)
		sd.activeQ.enq(newProc)
	}

	maxCostRunning := float32(0)
	for _, p := range sd.activeQ.getQ() {
		if p.willingToSpend() > maxCostRunning {
			maxCostRunning = p.willingToSpend()
		}
	}

	// if not from GS whose list we're in and change in mem is large, update the list
	if (float32(sd.memFree()) < 0.9*float32(ogMemFree)) && (sd.currHeapGSS >= 0) && (sd.currHeapGSS != fromGs) {
		heapToUse := sd.idleHeaps[sd.currHeapGSS]
		heapToUse.lock.Lock()
		// also if it is already in the heap, then replace it with the new value
		if contains(heapToUse.heap, sd.machineId) {
			remove(heapToUse.heap, sd.machineId)
		}
		toPush := TIdleMachine{
			machine:            sd.machineId,
			highestCostRunning: maxCostRunning,
			qlen:               sd.activeQ.qlen(),
			memAvail:           sd.memFree(),
		}
		heapToUse.heap.Push(toPush)
		heapToUse.lock.Unlock()
	}

	// don't want the GSS to take out idleness into account if we are already somewhere else
	dontWantToSendIdleInfo := (sd.currHeapGSS >= 0) && (sd.currHeapGSS != fromGs)
	if dontWantToSendIdleInfo {
		toWrite := fmt.Sprintf("    don't want to send; curr heap is actually %v \n", sd.currHeapGSS)
		logWrite(SCHED, toWrite)
		return false, TIdleMachine{}
	}

	stillNotIdle := (sd.currHeapGSS < 0) && !(sd.memFree() > IDLE_HEAP_MEM_THRESHOLD)
	if stillNotIdle {
		return false, TIdleMachine{}
	}

	wasButNowNoLongerIdle := (sd.currHeapGSS == fromGs) && (sd.memFree() < IDLE_HEAP_MEM_THRESHOLD)
	if wasButNowNoLongerIdle {
		sd.currHeapGSS = -1
	}

	newlyIdle := (sd.currHeapGSS < 0) && (sd.memFree() > IDLE_HEAP_MEM_THRESHOLD)
	if newlyIdle {
		sd.currHeapGSS = fromGs
	}

	return true, TIdleMachine{
		machine:            sd.machineId,
		highestCostRunning: maxCostRunning,
		qlen:               sd.activeQ.qlen(),
		memAvail:           sd.memFree(),
	}
}

// do numCores ticks of computation (only on procs in the activeQ)
func (sd *Machine) simulateRunProcs() {

	totalTicksLeftToGive := Tftick(sd.numCores)
	ticksLeftPerCore := make(map[int]Tftick, 0)
	coresWithTicksLeft := make(map[int]bool, 0)
	coresLeftThisRound := make(map[int]bool, 0)

	for i := 0; i < sd.numCores; i++ {
		ticksLeftPerCore[i] = Tftick(1)
		coresWithTicksLeft[i] = true
	}

	ogMemFree := sd.memFree()
	toWrite := fmt.Sprintf("%v, %v, %v", sd.worldNumProcsGenPerTick, int(*sd.currTickPtr), sd.machineId)
	logWrite(USAGE, toWrite)

	putProcOnCoreWithMaxTimeLeft := func() int {
		maxVal := Tftick(0.0)
		coreToUse := -1
		for i := 0; i < sd.numCores; i++ {
			if _, ok := coresLeftThisRound[i]; ok {
				if _, ok := coresWithTicksLeft[i]; ok {
					if ticksLeftPerCore[i] > maxVal {
						maxVal = ticksLeftPerCore[i]
						coreToUse = i
					}
				}
			}
		}
		delete(coresLeftThisRound, coreToUse)
		return coreToUse
	}

	toReq := make([]*Proc, 0)

	toWrite = fmt.Sprintf("\n==> %v @ %v, machine %v (on heap: %v, mem free: %v); has q: \n%v", sd.worldNumProcsGenPerTick, sd.currTickPtr.String(), sd.machineId, sd.currHeapGSS, sd.memFree(), sd.activeQ.SummaryString())
	logWrite(SCHED, toWrite)

	for sd.activeQ.qlen() > 0 && totalTicksLeftToGive-Tftick(TICK_SCHED_THRESHOLD) > 0.0 && len(coresWithTicksLeft) > 0 {

		for i := 0; i < sd.numCores; i++ {
			coresLeftThisRound[i] = true
		}

		// run by amount of money willing to spend
		coreToProc := make(map[int]*Proc, 0)
		for i := 0; i < sd.numCores; i++ {
			p := sd.activeQ.deq()
			if p == nil {
				continue
			}
			coreToUse := putProcOnCoreWithMaxTimeLeft()
			if coreToUse == -1 {
				sd.activeQ.enq(p)
				coreToProc[coreToUse] = nil
			} else {
				coreToProc[coreToUse] = p
			}
		}

		// run all the cores once
		for currCore := 0; currCore < sd.numCores; currCore++ {

			procToRun := coreToProc[currCore]

			if procToRun == nil {
				continue
			}

			toWrite := fmt.Sprintf("   core %v giving %v to proc %v \n", currCore, ticksLeftPerCore[currCore], procToRun.String())
			logWrite(SCHED, toWrite)

			ticksUsed, done := procToRun.runTillOutOrDone(ticksLeftPerCore[currCore])

			ticksLeftPerCore[currCore] -= ticksUsed
			totalTicksLeftToGive -= ticksUsed

			if ticksLeftPerCore[currCore] < TICK_SCHED_THRESHOLD {
				delete(coresWithTicksLeft, currCore)
			}

			if !done {
				toReq = append(toReq, procToRun)
			} else {
				// if the proc is done, update the ticksPassed to be exact for metrics etc
				procToRun.timeDone = *sd.currTickPtr + (1 - ticksLeftPerCore[currCore])

				toWrite := fmt.Sprintf("   -> done: %v\n", procToRun.String())
				logWrite(SCHED, toWrite)

				if (procToRun.timeDone - procToRun.timeStarted) > procToRun.compDone {
					toWrite := fmt.Sprintf("   ---> OVER %v \n", procToRun.String())
					logWrite(SCHED, toWrite)
				}

				toWrite = fmt.Sprintf("%v, %v, %v, %v \n", sd.worldNumProcsGenPerTick, procToRun.willingToSpend(), (procToRun.timeDone - procToRun.timeStarted).String(), procToRun.compDone.String())
				logWrite(PROCS_DONE, toWrite)
			}
		}

	}

	for _, p := range toReq {
		sd.activeQ.enq(p)
	}

	if totalTicksLeftToGive < 0.00002 {
		totalTicksLeftToGive = 0
	}
	toWrite = fmt.Sprintf(", %.3f, %v\n", float64(math.Copysign(float64(totalTicksLeftToGive), 1)), ogMemFree)
	logWrite(USAGE, toWrite)

	highestCost := float32(0)
	for _, p := range sd.activeQ.getQ() {
		if p.willingToSpend() > highestCost {
			highestCost = p.willingToSpend()
		}
	}

	if (sd.activeQ.qlen() > IDLE_HEAP_QLEN_THRESHOLD) && (sd.memFree() < IDLE_HEAP_MEM_THRESHOLD) {
		// are not idle
		if sd.currHeapGSS >= 0 {
			sd.idleHeaps[sd.currHeapGSS].lock.Lock()
			if contains(sd.idleHeaps[sd.currHeapGSS].heap, sd.machineId) {
				remove(sd.idleHeaps[sd.currHeapGSS].heap, sd.machineId)
			}
			sd.idleHeaps[sd.currHeapGSS].lock.Unlock()
			sd.currHeapGSS = -1
		}
		return
	}

	// only here if we are idle
	var heapToUse *IdleHeap
	if sd.currHeapGSS >= 0 {
		// already in a heap, need to just update that info
		heapToUse = sd.idleHeaps[sd.currHeapGSS]
	} else {
		// choose idle heap to use by power of k choices
		heapsToLookAt := pickRandomElementsMap(sd.idleHeaps, K_CHOICES_UP)

		minLength := math.MaxInt
		for gssId, possHeap := range heapsToLookAt {
			possHeap.lock.Lock()
			if possHeap.heap.Len() < minLength {
				minLength = possHeap.heap.Len()
				heapToUse = possHeap
				sd.currHeapGSS = gssId
			}
			possHeap.lock.Unlock()
		}
	}

	heapToUse.lock.Lock()
	// also if it is already in the heap, then replace it with the new value
	if contains(heapToUse.heap, sd.machineId) {
		remove(heapToUse.heap, sd.machineId)
	}
	toPush := TIdleMachine{
		machine:            sd.machineId,
		highestCostRunning: highestCost,
		qlen:               sd.activeQ.qlen(),
		memAvail:           sd.memFree(),
	}
	heapToUse.heap.Push(toPush)
	heapToUse.lock.Unlock()

}
