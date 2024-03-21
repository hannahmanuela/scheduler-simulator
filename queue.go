package slasched

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
		if currProc.effectiveSla() > p.effectiveSla() {
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

// gets Qs lowest priority proc
func (q *Queue) workSteal(maxMem Tmem) *Proc {
	if len(q.q) == 0 {
		return nil
	}

	// for i := len(q.q) - 1; i >= 0; i-- {
	// TODO: for now, don't steal procs under the threshold b/c maybe core running that proc just hasn't run yet
	for i := 0; i < len(q.q); i++ {
		if q.q[i].effectiveSla() > PUSH_SLA_THRESHOLD && Tmem(q.q[i].procTypeProfile.memUsg.avg) < maxMem {
			procSelected := q.q[i]
			newQ := append(q.q[:i], q.q[i+1:]...)
			q.q = newQ
			return procSelected
		}
	}
	return nil
}

func (q *Queue) remove(toRemove *Proc) {
	for i := 0; i < len(q.q); i++ {
		if q.q[i] == toRemove {
			newQ := append(q.q[:i], q.q[i+1:]...)
			q.q = newQ
			return
		}
	}
}

func (q *Queue) qlen() int {
	return len(q.q)
}
