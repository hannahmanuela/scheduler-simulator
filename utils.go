package slasched

import (
	"fmt"
	"math/rand"
)

type Ttick int
type Tmem int

type Tftick float64

func (f Tftick) String() string {
	return fmt.Sprintf("%.3fT", f)
}

type Website interface {
	genLoad(rand *rand.Rand) []*ProcInternals
}

func sampleNormal(mu, sigma float64) float64 {
	return rand.NormFloat64()*float64(sigma) + float64(mu)
}
