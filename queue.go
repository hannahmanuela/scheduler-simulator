package slasched

import (
	"math"
	"strconv"
)

// note: currently we are keeping queues ordered (by expected finishing "time")
type Queue struct {
	prioQs map[int][]*Proc
}

func newQueue() *Queue {
	q := &Queue{prioQs: make(map[int][]*Proc, 0)}
	return q
}

func (q *Queue) String() string {
	str := ""
	for prio, q := range q.prioQs {
		str += strconv.Itoa(prio) + ": "
		for _, p := range q {
			str += p.String() + "; "
		}
	}
	return str
}

func (q *Queue) getAllProcs() []*Proc {
	all := make([]*Proc, 0)
	for _, q := range q.prioQs {
		all = append(all, q...)
	}
	return all
}

func (q *Queue) enq(p *Proc) {

	if _, ok := q.prioQs[p.priority]; !ok {
		q.prioQs[p.priority] = []*Proc{p}
		return
	}

	q.prioQs[p.priority] = append(q.prioQs[p.priority], p)
	return
}

func (q *Queue) deq() *Proc {

	if len(q.prioQs) == 0 {
		return nil
	}

	minPrio := math.MaxInt
	for prio, _ := range q.prioQs {
		if prio < minPrio {
			minPrio = prio
		}
	}

	procSelected := q.prioQs[minPrio][0]
	q.prioQs[minPrio] = q.prioQs[minPrio][1:]
	return procSelected
}

func (q *Queue) remove(toRem *Proc) {
	newQueue := []*Proc{}

	for _, v := range q.prioQs[toRem.priority] {
		if v != toRem {
			newQueue = append(newQueue, v)
		}
	}

	q.prioQs[toRem.priority] = newQueue
}

func (q *Queue) numProcs() int {
	sum := 0
	for _, q := range q.prioQs {
		sum += len(q)
	}

	return sum
}
