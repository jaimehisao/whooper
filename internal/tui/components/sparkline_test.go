package components

import (
	"testing"
)

func TestSparkline_EmptyData(t *testing.T) {
	result := Sparkline(nil, 10)
	if result != "" {
		t.Errorf("expected empty string for empty data, got %q", result)
	}
}

func TestSparkline_SingleValue(t *testing.T) {
	result := Sparkline([]float64{42}, 10)
	if len([]rune(result)) != 1 {
		t.Errorf("expected 1 rune for single value, got %d runes: %q", len([]rune(result)), result)
	}
}

func TestSparkline_AllSameValues(t *testing.T) {
	data := []float64{5, 5, 5, 5}
	result := Sparkline(data, 10)
	runes := []rune(result)
	if len(runes) != 4 {
		t.Fatalf("expected 4 runes, got %d", len(runes))
	}
	// All values the same means span=0, forced to 1; normalized=0, so all should
	// map to the same block character
	first := runes[0]
	for i, r := range runes {
		if r != first {
			t.Errorf("index %d: expected %c, got %c", i, first, r)
		}
	}
}

func TestSparkline_Ascending(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5, 6, 7, 8}
	result := Sparkline(data, 10)
	runes := []rune(result)
	if len(runes) != 8 {
		t.Fatalf("expected 8 runes, got %d", len(runes))
	}

	blocks := []rune("▁▂▃▄▅▆▇█")
	// First should be lowest block, last should be highest
	if runes[0] != blocks[0] {
		t.Errorf("first rune should be %c (lowest), got %c", blocks[0], runes[0])
	}
	if runes[len(runes)-1] != blocks[len(blocks)-1] {
		t.Errorf("last rune should be %c (highest), got %c", blocks[len(blocks)-1], runes[len(runes)-1])
	}

	// Verify ascending order
	for i := 1; i < len(runes); i++ {
		if runes[i] < runes[i-1] {
			t.Errorf("expected ascending order at index %d: %c < %c", i, runes[i], runes[i-1])
		}
	}
}

func TestSparkline_WidthZero(t *testing.T) {
	result := Sparkline([]float64{1, 2, 3}, 0)
	if result != "" {
		t.Errorf("expected empty string for width 0, got %q", result)
	}
}
