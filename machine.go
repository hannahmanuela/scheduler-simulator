package slasched

import "fmt"

type Tmid int

type Ttickmap map[Tmid]Tftick
type Tprocmap map[Tmid]int

type Machine struct {
	mid      Tmid
	schedd   *Schedd
	numCores int
}

func newMachine(mid Tmid, numCores int) *Machine {
	sd := &Machine{
		mid:      mid,
		schedd:   newSchedd(numCores),
		numCores: numCores,
	}
	return sd
}

func (m Machine) String() string {
	str := fmt.Sprintf("mid: %d, schedd: %s, numCores: %d", m.mid, m.schedd.String(), m.numCores)
	return str
}
