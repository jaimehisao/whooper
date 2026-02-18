package views

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"git.infra.hisao.org/hisao/whooper/internal/models"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	"git.infra.hisao.org/hisao/whooper/internal/tui"
	"git.infra.hisao.org/hisao/whooper/internal/tui/components"
)

var sleepRanges = []int{7, 14, 30, 90}

type SleepModel struct {
	db       *store.DB
	rangeIdx int
	trend    []store.SleepTrendPoint
	sleeps   []models.Sleep
	loaded   bool
}

func NewSleep(db *store.DB) SleepModel {
	return SleepModel{db: db, rangeIdx: 2}
}

type sleepDataMsg struct {
	trend  []store.SleepTrendPoint
	sleeps []models.Sleep
}

func (m *SleepModel) Init() tea.Cmd {
	return m.Refresh()
}

func (m *SleepModel) Refresh() tea.Cmd {
	db := m.db
	rangeIdx := m.rangeIdx
	return func() tea.Msg {
		days := sleepRanges[rangeIdx]
		from := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour).Format("2006-01-02")
		trend, _ := db.GetSleepTrend(from, "")
		sleeps, _ := db.ListSleeps(from, "", true)
		return sleepDataMsg{trend: trend, sleeps: sleeps}
	}
}

func (m *SleepModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sleepDataMsg:
		m.trend = msg.trend
		m.sleeps = msg.sleeps
		m.loaded = true
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "<":
			if m.rangeIdx > 0 {
				m.rangeIdx--
				return m, m.Refresh()
			}
		case "right", ">":
			if m.rangeIdx < len(sleepRanges)-1 {
				m.rangeIdx++
				return m, m.Refresh()
			}
		}
	}
	return m, nil
}

func (m *SleepModel) View() string {
	if !m.loaded {
		return tui.MutedStyle.Render("Loading sleep data...")
	}

	days := sleepRanges[m.rangeIdx]
	header := tui.TitleStyle.Render(fmt.Sprintf("Sleep Trends (%dd)  < %dd >", days, days))

	if len(m.trend) == 0 {
		return header + "\n" + tui.MutedStyle.Render("No sleep data available.")
	}

	var sections []string
	sections = append(sections, header)

	var durations, efficiencies []float64
	for _, t := range m.trend {
		durations = append(durations, float64(t.DurationMilli)/3600000.0)
		efficiencies = append(efficiencies, t.EfficiencyPct)
	}

	durSpark := components.Sparkline(durations, 50)
	sections = append(sections, fmt.Sprintf("\n%s\n%s",
		tui.TextStyle.Render("Sleep Duration (hours)"),
		tui.AccentStyle.Render(durSpark)))

	effSpark := components.Sparkline(efficiencies, 50)
	sections = append(sections, fmt.Sprintf("\n%s\n%s",
		tui.TextStyle.Render("Sleep Efficiency (%%)"),
		tui.GreenStyle.Render(effSpark)))

	sections = append(sections, "\n"+tui.TitleStyle.Render("Sleep Stages"))

	limit := len(m.sleeps)
	if limit > 14 {
		limit = 14
	}

	lightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5B9BD5"))
	swsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4472C4"))
	remStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9B59B6"))
	awakeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E74C3C"))

	barWidth := 40
	for i := limit - 1; i >= 0; i-- {
		s := m.sleeps[i]
		if s.Score == nil {
			continue
		}
		ss := s.Score.StageSummary
		total := float64(ss.TotalInBedTimeMilli)
		if total <= 0 {
			continue
		}

		lightW := int(float64(ss.TotalLightSleepTimeMilli) / total * float64(barWidth))
		swsW := int(float64(ss.TotalSlowWaveSleepTimeMilli) / total * float64(barWidth))
		remW := int(float64(ss.TotalRemSleepTimeMilli) / total * float64(barWidth))
		awakeW := int(float64(ss.TotalAwakeTimeMilli) / total * float64(barWidth))
		remainder := barWidth - lightW - swsW - remW - awakeW
		lightW += remainder

		t, _ := time.Parse(time.RFC3339, s.Start)
		bar := awakeStyle.Render(repeatStr("█", awakeW)) +
			lightStyle.Render(repeatStr("█", lightW)) +
			swsStyle.Render(repeatStr("█", swsW)) +
			remStyle.Render(repeatStr("█", remW))

		hours := float64(ss.TotalInBedTimeMilli-ss.TotalAwakeTimeMilli) / 3600000.0
		sections = append(sections, fmt.Sprintf("  %s  %s  %.1fh",
			t.Format("Jan 02"), bar, hours))
	}

	sections = append(sections, fmt.Sprintf("\n  %s %s %s %s",
		awakeStyle.Render("█ Awake"),
		lightStyle.Render("█ Light"),
		swsStyle.Render("█ Deep"),
		remStyle.Render("█ REM")))

	if len(m.sleeps) > 0 && m.sleeps[0].Score != nil {
		s := m.sleeps[0]
		need := s.Score.SleepNeeded
		totalNeed := float64(need.BaselineMilli+need.NeedFromSleepDebtMilli+need.NeedFromRecentStrainMilli) / 3600000.0
		actual := float64(s.Score.StageSummary.TotalInBedTimeMilli-s.Score.StageSummary.TotalAwakeTimeMilli) / 3600000.0
		sections = append(sections, fmt.Sprintf("\n%s  Need: %.1fh  Actual: %.1fh",
			tui.TitleStyle.Render("Sleep Need vs Actual"),
			totalNeed, actual))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func repeatStr(s string, n int) string {
	if n <= 0 {
		return ""
	}
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
