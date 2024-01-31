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
	genLoad(nProcs int) []*ProcInternals
}

func sampleNormal(mu, sigma float64) float64 {
	return rand.NormFloat64()*float64(sigma) + float64(mu)
}

func findMaxIndex(numbers []float64) int {
	if len(numbers) == 0 {
		// Handle empty list case
		return -1
	}

	maxIndex := 0
	maxValue := numbers[0]

	for i, value := range numbers {
		if value > maxValue {
			maxIndex = i
			maxValue = value
		}
	}

	return maxIndex
}
