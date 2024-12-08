package slasched

type Ttenant struct {
	id                   Tid
	currNumCompTokens    Tftick
	numCompTokensPerTick int
	website              Website
}

func newTenant(numCompTokensPerTick int) *Ttenant {
	return &Ttenant{
		currNumCompTokens:    Tftick(numCompTokensPerTick),
		numCompTokensPerTick: numCompTokensPerTick,
		website:              newSimpleWebsite(),
	}
}

func (tn *Ttenant) genLoad(nProcsToGen int) []*ProcInternals {
	return tn.website.genLoad(nProcsToGen, tn.id)
}
