package slasched

import "fmt"

type Tmid int

type Ttickmap map[Tmid]Tftick
type Tprocmap map[Tmid]int

type Machine struct {
	mid      Tmid
	sched    *MachineSched
	numCores int
}

func newMachine(mid Tmid, numCores int) *Machine {
	sd := &Machine{
		mid:      mid,
		sched:    NewMachineSched(numCores),
		numCores: numCores,
	}
	return sd
}

func (m Machine) String() string {
	str := fmt.Sprintf("mid: %d, sched: %s, numCores: %d", m.mid, m.sched.String(), m.numCores)
	return str
}
