package slasched

import (
	"fmt"
	"math/rand"

	"golang.org/x/exp/constraints"
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

type Number interface {
	constraints.Integer | constraints.Float
}

func avg[T Number](list []T) float64 {
	if len(list) == 0 {
		return 0
	}

	var sum T
	sum = 0
	for _, val := range list {
		sum += val
	}
	return float64(sum) / float64(len(list))
}

func findMaxIndex(numbers []float64) int {
	if len(numbers) == 0 {
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
