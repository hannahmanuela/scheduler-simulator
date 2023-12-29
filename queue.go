package slasched

// note: currently we are keeping queues ordered (by expected finishing "time")
type Queue struct {
	q []*ProvProc
}

func newQueue() *Queue {
	q := &Queue{q: make([]*ProvProc, 0)}
	return q
}

func (q *Queue) String() string {
	str := ""
	for _, p := range q.q {
		str += p.String()
	}
	return str
}

func (q *Queue) enq(p *ProvProc) {

	if len(q.q) == 0 {
		q.q = append(q.q, p)
		return
	}

	for index, currProc := range q.q {
		if currProc.timeShouldBeDone > p.timeShouldBeDone {
			headQ := append(q.q[:index], p)
			tailQ := q.q[index:]
			q.q = append(headQ, tailQ...)
			return
		}
	}
	q.q = append(q.q, p)
}

func (q *Queue) deq() *ProvProc {
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
