package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestBarChart_Empty(t *testing.T) {
	result := BarChart(nil, 20)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestBarChart_SingleItem(t *testing.T) {
	items := []BarItem{
		{Label: "Test", Value: 50, MaxValue: 100, Color: lipgloss.Color("#FF0000")},
	}
	result := BarChart(items, 20)
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if strings.Contains(result, "\n") {
		t.Error("single item should not have newline")
	}
	if !strings.Contains(result, "50.0") {
		t.Error("should contain the value")
	}
}

func TestBarChart_MultipleItems(t *testing.T) {
	items := []BarItem{
		{Label: "A", Value: 100, MaxValue: 100, Color: lipgloss.Color("#FF0000")},
		{Label: "B", Value: 0, MaxValue: 100, Color: lipgloss.Color("#00FF00")},
	}
	result := BarChart(items, 10)
	lines := strings.Split(result, "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestBarChart_ZeroMaxValue(t *testing.T) {
	items := []BarItem{
		{Label: "X", Value: 5, MaxValue: 0, Color: lipgloss.Color("#FFF")},
	}
	// Should not panic with zero MaxValue
	result := BarChart(items, 10)
	if result == "" {
		t.Error("expected non-empty result")
	}
}
