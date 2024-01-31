package slasched

import "fmt"

type Tmid int

type Ttickmap map[Tmid]Tftick
type Tprocmap map[Tmid]int

type Machine struct {
	mid   Tmid
	sched *Sched
}

func newMachine(mid Tmid, lbConn chan *MachineMessages) *Machine {
	sd := &Machine{
		mid:   mid,
		sched: newSched(lbConn),
	}
	return sd
}

func (m Machine) String() string {
	str := fmt.Sprintf("mid: %d, sched: %s\n", m.mid, m.sched.String())
	return str
}
