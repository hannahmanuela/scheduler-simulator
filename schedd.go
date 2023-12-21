package main

import "fmt"

type Schedd struct {
	totMem Tmem
	q      *Queue
	util   float64
	ticks  Tftick
}

func newSchedd() *Schedd {
	sd := &Schedd{
		totMem: MAX_MEM,
		q:      newQueue(),
		ticks:  Tftick(0),
	}
	return sd
}

func (sd *Schedd) String() string {
	return fmt.Sprintf("{totMem %d memUsed %d q %v}", sd.totMem, sd.memUsed(), sd.q)
}

func (sd *Schedd) memUsed() Tmem {
	return sd.q.memUsed()
}

func (sd *Schedd) run() {
	if len(sd.q.q) == 0 {
		return
	}
	sd.util += float64(1)
	// TODO: does this make sense?
	ticks := sd.q.run()
	sd.ticks += ticks
}

// TODO: firgure out what this is doing
func (sd *Schedd) zap() bool {
	proc := -1
	m := Tmem(0)
	for i, p := range sd.q.q {
		if p.memUsed > m {
			proc = i
			m = p.memUsed
		}
	}
	if proc != -1 {
		sd.q.zap(proc)
		return true
	}
	return false
}
