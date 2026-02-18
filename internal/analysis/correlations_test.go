package analysis

import (
	"math"
	"testing"
)

func TestPearsonCorrelation_PerfectPositive(t *testing.T) {
	x := []float64{1, 2, 3}
	y := []float64{2, 4, 6}
	r := PearsonCorrelation(x, y)
	if math.Abs(r-1.0) > 1e-9 {
		t.Errorf("expected r = 1.0 for perfect positive correlation, got %f", r)
	}
}

func TestPearsonCorrelation_PerfectNegative(t *testing.T) {
	x := []float64{1, 2, 3}
	y := []float64{6, 4, 2}
	r := PearsonCorrelation(x, y)
	if math.Abs(r-(-1.0)) > 1e-9 {
		t.Errorf("expected r = -1.0 for perfect negative correlation, got %f", r)
	}
}

func TestPearsonCorrelation_NoCorrelation(t *testing.T) {
	// Symmetric data with no linear relationship
	x := []float64{1, 2, 3, 4, 5}
	y := []float64{2, 4, 1, 4, 2}
	r := PearsonCorrelation(x, y)
	if math.Abs(r) > 0.3 {
		t.Errorf("expected r close to 0 for uncorrelated data, got %f", r)
	}
}

func TestPearsonCorrelation_EmptySlices(t *testing.T) {
	r := PearsonCorrelation(nil, nil)
	if r != 0 {
		t.Errorf("expected 0 for empty slices, got %f", r)
	}
}

func TestPearsonCorrelation_UnequalLengths(t *testing.T) {
	x := []float64{1, 2, 3}
	y := []float64{1, 2}
	r := PearsonCorrelation(x, y)
	if r != 0 {
		t.Errorf("expected 0 for unequal lengths, got %f", r)
	}
}

func TestPearsonCorrelation_AllSameValues(t *testing.T) {
	x := []float64{5, 5, 5}
	y := []float64{3, 3, 3}
	r := PearsonCorrelation(x, y)
	if r != 0 {
		t.Errorf("expected 0 when denominator is 0 (constant values), got %f", r)
	}
}
