package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Gauge renders a visual gauge bar for a recovery score (0-100).
// The bar is colored based on recovery zone: green >= 67, yellow >= 34, red < 34.
func Gauge(score float64, width int) string {
	if width <= 0 {
		width = 30
	}

	var color lipgloss.Color
	switch {
	case score >= 67:
		color = lipgloss.Color("#00D46A")
	case score >= 34:
		color = lipgloss.Color("#FFCC00")
	default:
		color = lipgloss.Color("#FF3333")
	}

	pct := score / 100.0
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}

	filled := int(pct * float64(width))
	empty := width - filled

	barStyle := lipgloss.NewStyle().Foreground(color)
	coloredStyle := lipgloss.NewStyle().Foreground(color).Bold(true)

	label := coloredStyle.Render(fmt.Sprintf("Recovery: %.0f%%", score))
	bar := "[" + barStyle.Render(strings.Repeat("█", filled)) + strings.Repeat("░", empty) + "]"

	return fmt.Sprintf("%s\n%s", label, bar)
}
