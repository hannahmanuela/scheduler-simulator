package slasched

import (
	"math"
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
		str += p.String() + "; "
	}
	return str
}

func (q *Queue) getQ() []*Proc {
	return q.q
}

func (q *Queue) enq(p *Proc) {
	if len(q.q) == 0 {
		q.q = append(q.q, p)
		return
	}

	for index, currProc := range q.q {
		if currProc.getRelDeadline() > p.getRelDeadline() {
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

func (q *Queue) getHOLSlack(currTime Tftick) Tftick {

	if len(q.q) == 0 {
		return Tftick(1000)
	}

	runningWaitTime := Tftick(0)
	headSlack := q.q[0].getSlack(currTime)
	extraSlack := Tftick(0)

	for _, p := range q.q {
		currExtra := p.getSlack(currTime) - runningWaitTime
		if currExtra < extraSlack {
			extraSlack = currExtra
		}
		runningWaitTime += p.getMaxCompLeft()
	}

	holSlack := Tftick(math.Min(float64(headSlack), float64(extraSlack)))

	return holSlack
}
