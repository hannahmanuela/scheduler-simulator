package slasched

import "fmt"

type Tmid int

type Ttickmap map[Tmid]Tftick
type Tprocmap map[Tmid]int

type SchedulerType int

const (
	SHINJUKU SchedulerType = iota
	PS
	EDF
)

type Machine struct {
	mid       Tmid
	sched     Sched
	schedType SchedulerType
}

func newMachine(schedType SchedulerType, mid Tmid, lbConn chan *MachineMessages) *Machine {
	m := &Machine{
		mid:       mid,
		schedType: schedType,
	}
	switch schedType {
	case SHINJUKU:
		m.sched = newShinjukuSched(lbConn, mid)
	case PS:
		m.sched = newPSSched(lbConn, mid)
	case EDF:
		m.sched = newEDFSched(lbConn, mid)
	}
	return m
}

func (m Machine) String() string {
	str := fmt.Sprintf("mid: %d, sched: %s\n", m.mid, m.sched.String())
	return str
}
