package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// BarItem represents a single bar in a horizontal bar chart.
type BarItem struct {
	Label    string
	Value    float64
	MaxValue float64
	Color    lipgloss.Color
}

// BarChart renders a horizontal bar chart from the given items.
// barWidth controls the maximum width of the bar portion.
func BarChart(items []BarItem, barWidth int) string {
	if len(items) == 0 {
		return ""
	}

	// Find the longest label for alignment.
	maxLabel := 0
	for _, item := range items {
		if len(item.Label) > maxLabel {
			maxLabel = len(item.Label)
		}
	}

	var sb strings.Builder
	for i, item := range items {
		maxVal := item.MaxValue
		if maxVal <= 0 {
			maxVal = 1
		}
		filled := int(item.Value / maxVal * float64(barWidth))
		if filled < 0 {
			filled = 0
		}
		if filled > barWidth {
			filled = barWidth
		}
		empty := barWidth - filled

		barStyle := lipgloss.NewStyle().Foreground(item.Color)
		bar := barStyle.Render(strings.Repeat("█", filled)) + strings.Repeat("░", empty)

		label := fmt.Sprintf("%-*s", maxLabel, item.Label)
		value := fmt.Sprintf("%6.1f", item.Value)

		sb.WriteString(fmt.Sprintf("%s %s %s", label, bar, value))
		if i < len(items)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
