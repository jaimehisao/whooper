package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Table renders a simple formatted table with highlighted headers.
// columnWidths controls the width of each column.
// If a column width is 0, it defaults to 12.
func Table(headers []string, rows [][]string, columnWidths []int) string {
	if len(headers) == 0 {
		return ""
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#16A085"))

	mutedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))

	var sb strings.Builder

	// Render header row.
	var headerCells []string
	for i, h := range headers {
		w := colWidth(columnWidths, i)
		headerCells = append(headerCells, headerStyle.Render(fmt.Sprintf("%-*s", w, truncate(h, w))))
	}
	sb.WriteString(strings.Join(headerCells, " "))
	sb.WriteString("\n")

	// Separator.
	var sepParts []string
	for i := range headers {
		w := colWidth(columnWidths, i)
		sepParts = append(sepParts, mutedStyle.Render(strings.Repeat("─", w)))
	}
	sb.WriteString(strings.Join(sepParts, " "))
	sb.WriteString("\n")

	// Render data rows.
	for _, row := range rows {
		var cells []string
		for i := range headers {
			w := colWidth(columnWidths, i)
			val := ""
			if i < len(row) {
				val = row[i]
			}
			cells = append(cells, fmt.Sprintf("%-*s", w, truncate(val, w)))
		}
		sb.WriteString(strings.Join(cells, " "))
		sb.WriteString("\n")
	}

	return sb.String()
}

// HighlightedTable renders a table where one row (cursor) is highlighted.
func HighlightedTable(headers []string, rows [][]string, columnWidths []int, cursor int) string {
	if len(headers) == 0 {
		return ""
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#16A085"))

	mutedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E0E0E0")).
		Background(lipgloss.Color("#16A085"))

	var sb strings.Builder

	// Render header row.
	var headerCells []string
	for i, h := range headers {
		w := colWidth(columnWidths, i)
		headerCells = append(headerCells, headerStyle.Render(fmt.Sprintf("%-*s", w, truncate(h, w))))
	}
	sb.WriteString(strings.Join(headerCells, " "))
	sb.WriteString("\n")

	// Separator.
	var sepParts []string
	for i := range headers {
		w := colWidth(columnWidths, i)
		sepParts = append(sepParts, mutedStyle.Render(strings.Repeat("─", w)))
	}
	sb.WriteString(strings.Join(sepParts, " "))
	sb.WriteString("\n")

	// Render data rows.
	for r, row := range rows {
		var cells []string
		for i := range headers {
			w := colWidth(columnWidths, i)
			val := ""
			if i < len(row) {
				val = row[i]
			}
			cell := fmt.Sprintf("%-*s", w, truncate(val, w))
			if r == cursor {
				cell = selectedStyle.Render(cell)
			}
			cells = append(cells, cell)
		}
		sb.WriteString(strings.Join(cells, " "))
		sb.WriteString("\n")
	}

	return sb.String()
}

func colWidth(widths []int, i int) int {
	if i < len(widths) && widths[i] > 0 {
		return widths[i]
	}
	return 12
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
