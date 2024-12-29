package slasched

import (
	"math"
	"strconv"
)

// note: currently we are keeping queues ordered (by expected finishing "time")
type Queue struct {
	q []*Proc
}

func newQueue() *Queue {
	q := &Queue{q: make([]*Proc, 0)}
	return q
}

func (q *Queue) String() string {
	str := ""
	for _, p := range q.q {
		str += p.String() + ";\n"
	}
	return str
}

func (q *Queue) SummaryString() string {
	numPerPrice := make(map[float32]int, N_PRIORITIES)

	for prio := 0; prio < N_PRIORITIES; prio++ {
		numPerPrice[mapPriorityToDollars(prio)] = 0
	}

	for _, p := range q.q {
		numPerPrice[p.willingToSpend()] += 1
	}

	str := ""
	for prio := 0; prio < N_PRIORITIES; prio++ {
		str += strconv.FormatFloat(float64(mapPriorityToDollars(prio)), 'f', 3, 32) + ": " + strconv.Itoa(numPerPrice[mapPriorityToDollars(prio)]) + "\n"
	}

	return str
}

func (q *Queue) getQ() []*Proc {
	return q.q
}

func (q *Queue) peek() *Proc {
	if len(q.q) == 0 {
		return nil
	}

	return q.q[0]
}

func (q *Queue) enq(p *Proc) {
	if len(q.q) == 0 {
		q.q = append(q.q, p)
		return
	}

	for index, currProc := range q.q {
		if p.willingToSpend() > currProc.willingToSpend() ||
			((currProc.willingToSpend() == p.willingToSpend()) && p.timePlaced < currProc.timePlaced) ||
			((currProc.willingToSpend() == p.willingToSpend()) && (currProc.timePlaced == p.timePlaced) && p.compDone > currProc.compDone) {
			q.q = append(q.q[:index+1], q.q[index:]...)
			q.q[index] = p
			return
		}
	}
	q.q = append(q.q, p)
}

func (q *Queue) deq() *Proc {
	if len(q.q) == 0 {
		return nil
	}
	procSelected := q.q[0]
	q.q = q.q[1:]
	return procSelected
}

func (q *Queue) qlen() int {
	return len(q.q)
}

// func (q *Queue) remove(procToRemove *Proc) {

// 	newQ := make([]*Proc, len(q.q)-1)

// 	for i, p := range q.q {
// 		if p == procToRemove {
// 			newQ = append(q.q[:i], q.q[i+1:]...)
// 		}
// 	}

// 	q.q = newQ

// }

func (q *Queue) checkKill(newProc *Proc) (Tid, float32) {

	minMoneyThrownAway := float32(math.MaxFloat32)
	procId := Tid(-1)

	for _, p := range q.q {
		if (p.maxMem() > newProc.maxMem()) && (p.willingToSpend() < newProc.willingToSpend()) {
			wldThrow := float32(float32(p.compDone) * p.willingToSpend())
			if wldThrow < minMoneyThrownAway {
				procId = p.procId
				minMoneyThrownAway = wldThrow
			}
		}
	}

	return procId, minMoneyThrownAway

}

func (q *Queue) kill(pid Tid) {

	tmp := make([]*Proc, 0)

	for _, currProc := range q.q {
		if currProc.procId != pid {
			tmp = append(tmp, currProc)
		}
	}

	q.q = tmp

}

type MultiQueue struct {
	qMap map[float32]*Queue
}

func NewMultiQ() MultiQueue {
	mq := MultiQueue{
		qMap: make(map[float32]*Queue, N_PRIORITIES),
	}

	for prio := 0; prio < N_PRIORITIES; prio++ {
		price := mapPriorityToDollars(prio)
		mq.qMap[price] = newQueue()
	}
	return mq
}

func (mq MultiQueue) String() string {
	str := ""
	for prio := 0; prio < N_PRIORITIES; prio++ {
		price := mapPriorityToDollars(prio)
		if _, ok := mq.qMap[price]; !ok {
			str += strconv.FormatFloat(float64(price), 'f', 3, 32) + mq.qMap[price].String() + ";\n"
		}
	}
	return str
}

func (mq MultiQueue) len() int {
	totalLen := 0
	for prio := 0; prio < N_PRIORITIES; prio++ {
		totalLen += mq.qMap[mapPriorityToDollars(prio)].qlen()
	}
	return totalLen
}

func (mq MultiQueue) deq(currTick Tftick) *Proc {

	minRatio := float32(math.MaxFloat32)
	bestPrice := float32(-1)

	for prio := 0; prio < N_PRIORITIES; prio++ {
		price := mapPriorityToDollars(prio)
		q := mq.qMap[price]
		// ratio
		headProc := q.peek()
		if headProc == nil {
			continue
		}
		ratio := float32(currTick-headProc.timeStarted) / headProc.willingToSpend()
		if ratio <= minRatio {
			minRatio = ratio
			bestPrice = price
		}
	}

	if bestPrice < 0 {
		return nil
	}

	return mq.qMap[bestPrice].deq()
}

func (mq MultiQueue) enq(proc *Proc) {

	mq.qMap[proc.willingToSpend()].enq(proc)

}
