package slasched

import "math"

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
