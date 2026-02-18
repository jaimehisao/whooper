package analysis

import (
	"math"
	"testing"
)

func TestMovingAverage_EmptyData(t *testing.T) {
	result := MovingAverage(nil, 3)
	if result != nil {
		t.Errorf("expected nil for empty data, got %v", result)
	}
}

func TestMovingAverage_WindowZero(t *testing.T) {
	result := MovingAverage([]float64{1, 2, 3}, 0)
	if result != nil {
		t.Errorf("expected nil for window 0, got %v", result)
	}
}

func TestMovingAverage_WindowOne(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5}
	result := MovingAverage(data, 1)
	if len(result) != len(data) {
		t.Fatalf("expected length %d, got %d", len(data), len(result))
	}
	for i, v := range data {
		if result[i] != v {
			t.Errorf("index %d: expected %f, got %f", i, v, result[i])
		}
	}
}

func TestMovingAverage_WindowThree(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5}
	result := MovingAverage(data, 3)

	// i=0: sum=1, w=1 -> 1/1 = 1.0
	// i=1: sum=3, w=2 -> 3/2 = 1.5
	// i=2: sum=6, w=3 -> 6/3 = 2.0
	// i=3: sum=9, w=3 -> 9/3 = 3.0
	// i=4: sum=12, w=3 -> 12/3 = 4.0
	expected := []float64{1.0, 1.5, 2.0, 3.0, 4.0}

	if len(result) != len(expected) {
		t.Fatalf("expected length %d, got %d", len(expected), len(result))
	}
	for i, want := range expected {
		if math.Abs(result[i]-want) > 1e-9 {
			t.Errorf("index %d: expected %f, got %f", i, want, result[i])
		}
	}
}

func TestMovingAverage_WindowLargerThanData(t *testing.T) {
	data := []float64{2, 4}
	result := MovingAverage(data, 10)
	if len(result) != 2 {
		t.Fatalf("expected length 2, got %d", len(result))
	}
	// i=0: sum=2, w=1 -> 2.0
	// i=1: sum=6, w=2 -> 3.0
	if math.Abs(result[0]-2.0) > 1e-9 {
		t.Errorf("index 0: expected 2.0, got %f", result[0])
	}
	if math.Abs(result[1]-3.0) > 1e-9 {
		t.Errorf("index 1: expected 3.0, got %f", result[1])
	}
}

func TestPercentChange_Empty(t *testing.T) {
	if result := PercentChange(nil); result != nil {
		t.Errorf("expected nil for empty data, got %v", result)
	}
}

func TestPercentChange_Single(t *testing.T) {
	if result := PercentChange([]float64{42}); result != nil {
		t.Errorf("expected nil for single element, got %v", result)
	}
}

func TestPercentChange_Values(t *testing.T) {
	data := []float64{100, 110, 99}
	result := PercentChange(data)

	// (110-100)/100*100 = 10
	// (99-110)/110*100 = -10
	expected := []float64{10, -10}

	if len(result) != len(expected) {
		t.Fatalf("expected length %d, got %d", len(expected), len(result))
	}
	for i, want := range expected {
		if math.Abs(result[i]-want) > 0.01 {
			t.Errorf("index %d: expected %f, got %f", i, want, result[i])
		}
	}
}

func TestPercentChange_ZeroValue(t *testing.T) {
	data := []float64{0, 5, 10}
	result := PercentChange(data)

	if len(result) != 2 {
		t.Fatalf("expected length 2, got %d", len(result))
	}
	// 0 in denominator should return 0, not panic
	if result[0] != 0 {
		t.Errorf("expected 0 for zero denominator, got %f", result[0])
	}
	// (10-5)/5*100 = 100
	if math.Abs(result[1]-100.0) > 1e-9 {
		t.Errorf("expected 100.0, got %f", result[1])
	}
}
