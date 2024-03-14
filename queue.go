package slasched

import "sync"

// note: currently we are keeping queues ordered (by expected finishing "time")
type Queue struct {
	q []*Proc
	m sync.RWMutex
}

func newQueue() *Queue {
	q := &Queue{q: make([]*Proc, 0)}
	return q
}

func (q *Queue) String() string {
	q.m.RLock()
	defer q.m.RUnlock()

	str := ""
	for _, p := range q.q {
		str += p.String()
	}
	return str
}

func (q *Queue) getQ() []*Proc {
	q.m.RLock()
	defer q.m.RUnlock()

	return q.q
}

func (q *Queue) enq(p *Proc) {
	q.m.Lock()
	defer q.m.Unlock()

	if len(q.q) == 0 {
		q.q = append(q.q, p)
		return
	}

	for index, currProc := range q.q {
		if currProc.timeShouldBeDone > p.timeShouldBeDone {
			q.q = append(q.q[:index+1], q.q[index:]...)
			q.q[index] = p
			return
		}
	}
	q.q = append(q.q, p)
}

func (q *Queue) deq() *Proc {
	q.m.Lock()
	defer q.m.Unlock()

	if len(q.q) == 0 {
		return nil
	}
	procSelected := q.q[0]
	q.q = q.q[1:]
	return procSelected
}

func (q *Queue) qlen() int {
	q.m.RLock()
	defer q.m.RUnlock()

	return len(q.q)
}
