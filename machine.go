package slasched

import "fmt"

type Tmid int

type Ttickmap map[Tmid]Tftick
type Tprocmap map[Tmid]int

type Machine struct {
	mid    Tmid
	schedd *Schedd
}

func newMachine(mid Tmid) *Machine {
	sd := &Machine{
		mid:    mid,
		schedd: newSchedd(),
	}
	return sd
}

func (m Machine) String() string {
	str := fmt.Sprintf("mid: %d, schedd: %s", m.mid, m.schedd.String())
	return str
}
