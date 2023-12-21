package main

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
	genLoad(rand *rand.Rand) []*Proc
}

func zipf(r *rand.Rand) uint64 {
	z := rand.NewZipf(r, 2.0, 1.0, MAX_SERVICE_TIME-1)
	return z.Uint64() + 1
}
