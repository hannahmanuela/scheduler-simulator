package main

import (
	"fmt"
)

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
	q.q = append(q.q, p)
}

// TODO: find a proc to run
func (q *Queue) deq(n Tmem) *Proc {
	if len(q.q) == 0 {
		return nil
	}
	procSelected := q.q[0]
	q.q = q.q[1:]
	return procSelected
}

func (q *Queue) find(n Tmem) *Proc {
	for i, p := range q.q {
		if p.nMem <= n {
			q.q = append(q.q[0:i], q.q[i+1:]...)
			return p
		}
	}
	return nil
}

func (q *Queue) zap(proc int) {
	q.q = append(q.q[0:proc], q.q[proc+1:]...)
}

// spreads 1 tick around the procs according to its prioritization
func (q *Queue) run() {
	// TODO: maybe have queue be sorted by closeness of ticksPassed to SLA?
	n := Tftick(float64(1.0 / float64(len(q.q))))
	for n > 0 && len(q.q) > 0 {
		newProcQ := make([]*Proc, 0)
		ticksLeftOver := Tftick(0)
		for _, currProc := range q.q {
			// TODO: redefine u better
			u := n
			ticksUsed, done := currProc.runTillOutOrDone(u)
			ticksLeftOver = n - ticksUsed
			if !done {
				newProcQ = append(newProcQ, currProc)
			}
		}
		q.q = newProcQ
		if len(q.q) > 0 {
			n = Tftick(float64(ticksLeftOver) / float64(len(q.q)))
			if n > 0.001 {
				fmt.Printf("another round of scheduling %v\n", n)
			} else {
				n = Tftick(0)
			}
		}
	}

	// add 1 to all procs ticksPassed
	for _, currProc := range q.q {
		currProc.ticksPassed += 1
	}

}

func (q *Queue) memUsed() Tmem {
	m := Tmem(0)
	for _, p := range q.q {
		m += p.memUsed
	}
	return m
}

func (q *Queue) qlen() int {
	return len(q.q)
}
