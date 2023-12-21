package main

import "fmt"

type Tmid int

type Ttickmap map[Tmid]Tftick
type Tprocmap map[Tmid]int

type Machine struct {
	mid    Tmid
	schedd *Schedd
	nProcs int
	procs  []*Proc
}

func newMachine(mid Tmid) *Machine {
	sd := &Machine{
		mid:    mid,
		schedd: newSchedd(),
		nProcs: 0,
		procs:  make([]*Proc, 0),
	}
	return sd
}

func (m Machine) String() string {
	str := fmt.Sprintf("num procs running: %d, schedd: %s", m.nProcs, m.schedd.String())
	return str
}

// TODO:
func (m Machine) util() float64 {
	return 0
}
