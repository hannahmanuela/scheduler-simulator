package slasched

type Ttenant struct {
	id      Tid
	website Website
}

func newTenant() *Ttenant {
	return &Ttenant{
		website: newSimpleWebsite(),
	}
}

func (tn *Ttenant) genLoad(nProcsToGen int) []*ProcInternals {
	return tn.website.genLoad(nProcsToGen, tn.id)
}
