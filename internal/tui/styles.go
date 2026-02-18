package tui

import "github.com/charmbracelet/lipgloss"

var (
	ColorGreen  = lipgloss.Color("#00D46A")
	ColorYellow = lipgloss.Color("#FFCC00")
	ColorRed    = lipgloss.Color("#FF3333")
	ColorBg     = lipgloss.Color("#1A1A2E")
	ColorText   = lipgloss.Color("#E0E0E0")
	ColorAccent = lipgloss.Color("#16A085")
	ColorMuted  = lipgloss.Color("#666666")

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent).
			MarginBottom(1)

	TabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(ColorMuted)

	ActiveTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(ColorText).
			Background(ColorAccent).
			Bold(true)

	GreenStyle = lipgloss.NewStyle().
			Foreground(ColorGreen)

	YellowStyle = lipgloss.NewStyle().
			Foreground(ColorYellow)

	RedStyle = lipgloss.NewStyle().
			Foreground(ColorRed)

	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorAccent).
			Padding(1, 2)

	AccentStyle = lipgloss.NewStyle().
			Foreground(ColorAccent)

	TextStyle = lipgloss.NewStyle().
			Foreground(ColorText)
)

// RecoveryColor returns the appropriate color style based on recovery score.
func RecoveryColor(score float64) lipgloss.Style {
	switch {
	case score >= 67:
		return GreenStyle
	case score >= 34:
		return YellowStyle
	default:
		return RedStyle
	}
}
