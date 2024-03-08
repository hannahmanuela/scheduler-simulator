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

func newMachine(mid Tid, lbConn chan *MachineMessages, numCores int) *Machine {
	m := &Machine{
		mid:      mid,
		numCores: numCores,
		sched:    newSched(lbConn, mid, numCores),
	}
	return m
}

func (m Machine) String() string {
	str := fmt.Sprintf("mid: %d, sched: %s\n", m.mid, m.sched.String())
	return str
}
