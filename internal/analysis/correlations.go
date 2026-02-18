package analysis

import "math"

// PearsonCorrelation computes Pearson's r for two equal-length slices.
func PearsonCorrelation(x, y []float64) float64 {
	n := len(x)
	if n == 0 || n != len(y) {
		return 0
	}

	var sumX, sumY, sumXY, sumX2, sumY2 float64
	for i := 0; i < n; i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
		sumY2 += y[i] * y[i]
	}

	nf := float64(n)
	num := nf*sumXY - sumX*sumY
	den := math.Sqrt((nf*sumX2 - sumX*sumX) * (nf*sumY2 - sumY*sumY))
	if den == 0 {
		return 0
	}
	return num / den
}
