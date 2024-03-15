package slasched

import "fmt"

type Tid int

type Ttickmap map[Tid]Tftick
type Tprocmap map[Tid]int

type Machine struct {
	mid      Tid
	sched    *Sched
	numCores int
}

func newMachine(mid Tid, numCores int, lbConnSend chan *Message, lbConnRecv chan *Message) *Machine {
	sd := &Machine{
		mid:      mid,
		sched:    newSched(lbConnSend, lbConnRecv, mid, numCores),
		numCores: numCores,
	}
	return sd
}

func (m *Machine) String() string {
	str := fmt.Sprintf("mid: %d, sched: %s\n", m.mid, m.sched.String())
	return str
}
