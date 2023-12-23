package slasched

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
		str += p.String()
	}
	return str
}

func (q *Queue) enq(p *Proc) {

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

func (q *Queue) deq() *Proc {
	if len(q.q) == 0 {
		return nil
	}
	procSelected := q.q[0]
	q.q = q.q[1:]
	return procSelected
}

func (q *Queue) finishProc(index int) {
	q.q = append(q.q[:index], q.q[index+1:]...)
}

func (q *Queue) qlen() int {
	return len(q.q)
}
