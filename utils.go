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
	genLoad() []*ProcInternals
}

func sampleNormal(mu, sigma float64) float64 {
	return rand.NormFloat64()*float64(sigma) + float64(mu)
}

func sampleFromWeightList(weights []float64) int {
	if len(weights) == 0 {
		return -1
	}

	totalWeight := 0.0
	cdf := make([]float64, 0)
	for _, weight := range weights {
		totalWeight += weight
		cdf = append(cdf, totalWeight)
	}

	// Generate a random number between 0 and totalWeight
	randomNumber := rand.Float64() * totalWeight

	// Find the index of the selected item
	selectedIndex := 0
	for randomNumber > cdf[selectedIndex] {
		selectedIndex++
	}

	return selectedIndex
}

func normalizeList(values []float64, maxValue float64) []float64 {
	maxOriginal := values[0]
	for _, value := range values {
		if value > maxOriginal {
			maxOriginal = value
		}
	}

	normalizationFactor := maxValue / maxOriginal

	normalizedValues := make([]float64, len(values))
	for i, value := range values {
		normalizedValues[i] = value * normalizationFactor
	}

	return normalizedValues
}
