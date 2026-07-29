package views

import (
	"fmt"
	"strings"
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
	width    int
	rangeIdx int
	trend    []store.SleepTrendPoint
	sleeps   []models.Sleep
	loaded   bool
	err      string
}

func NewSleep(db *store.DB) SleepModel {
	return SleepModel{db: db, rangeIdx: 2, width: 80}
}

type sleepDataMsg struct {
	trend  []store.SleepTrendPoint
	sleeps []models.Sleep
	err    string
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
		msg := sleepDataMsg{}
		var errs []string

		trend, err := db.GetSleepTrend(from, "")
		if err != nil {
			errs = append(errs, err.Error())
		}
		msg.trend = trend

		sleeps, err := db.ListSleeps(from, "", true)
		if err != nil {
			errs = append(errs, err.Error())
		}
		msg.sleeps = sleeps

		if len(errs) > 0 {
			msg.err = fmt.Sprintf("sleep: %s", errs)
		}
		return msg
	}
}

func (m *SleepModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sleepDataMsg:
		m.trend = msg.trend
		m.sleeps = msg.sleeps
		m.err = msg.err
		m.loaded = true
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
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
	sparkW := m.sparkWidth()
	barW := sparkW - 10
	if barW < 20 {
		barW = 20
	}

	sections := make([]string, 0, 24)
	sections = append(sections, header)

	if m.err != "" {
		sections = append(sections, tui.RedStyle.Render("Error: "+m.err))
	}

	if len(m.trend) == 0 {
		sections = append(sections, tui.MutedStyle.Render("No sleep data available. Run 'whooper sync' first."))
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}

	durations := make([]float64, 0, len(m.trend))
	efficiencies := make([]float64, 0, len(m.trend))
	performances := make([]float64, 0, len(m.trend))
	consistencies := make([]float64, 0, len(m.trend))
	for _, t := range m.trend {
		durations = append(durations, float64(t.DurationMilli)/3600000.0)
		efficiencies = append(efficiencies, t.EfficiencyPct)
		performances = append(performances, t.PerformancePct)
		consistencies = append(consistencies, t.ConsistencyPct)
	}

	summary := sleepSummary(m.trend, m.sleeps)
	sections = append(sections, fmt.Sprintf(
		"\nAvg: %.1fh  Need gap: %+.1fh  Performance: %.0f%%  Efficiency: %.0f%%  Consistency: %.0f%%  Dist/night: %.0f",
		summary.avgActualHours,
		summary.avgGapHours,
		summary.avgPerformancePct,
		summary.avgEfficiencyPct,
		summary.avgConsistencyPct,
		summary.avgDisturbances,
	))

	durSpark := components.Sparkline(durations, sparkW)
	sections = append(sections, fmt.Sprintf("\n%s\n%s",
		tui.TextStyle.Render("Sleep Duration (hours)"),
		tui.AccentStyle.Render(durSpark)))

	needGaps := sleepNeedGaps(m.sleeps)
	if len(needGaps) > 0 {
		gapSpark := components.Sparkline(needGaps, sparkW)
		sections = append(sections, fmt.Sprintf("\n%s\n%s",
			tui.TextStyle.Render("Sleep Need Gap (actual - need)"),
			tui.AccentStyle.Render(gapSpark)))
	}

	perfSpark := components.Sparkline(performances, sparkW)
	sections = append(sections, fmt.Sprintf("\n%s\n%s",
		tui.TextStyle.Render("Sleep Performance (%%)"),
		tui.GreenStyle.Render(perfSpark)))

	effSpark := components.Sparkline(efficiencies, sparkW)
	sections = append(sections, fmt.Sprintf("\n%s\n%s",
		tui.TextStyle.Render("Sleep Efficiency (%%)"),
		tui.GreenStyle.Render(effSpark)))

	consSpark := components.Sparkline(consistencies, sparkW)
	sections = append(sections, fmt.Sprintf("\n%s\n%s",
		tui.TextStyle.Render("Sleep Consistency (%%)"),
		tui.AccentStyle.Render(consSpark)))

	table := recentSleepTable(m.sleeps, 7)
	if table != "" {
		sections = append(sections, "\n"+tui.TitleStyle.Render("Recent Nights"))
		sections = append(sections, table)
	}

	sections = append(sections, "\n"+tui.TitleStyle.Render("Sleep Stages"))

	limit := len(m.sleeps)
	if limit > 14 {
		limit = 14
	}

	lightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5B9BD5"))
	swsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4472C4"))
	remStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9B59B6"))
	awakeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E74C3C"))

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

		lightW := int(float64(ss.TotalLightSleepTimeMilli) / total * float64(barW))
		swsW2 := int(float64(ss.TotalSlowWaveSleepTimeMilli) / total * float64(barW))
		remW := int(float64(ss.TotalRemSleepTimeMilli) / total * float64(barW))
		awakeW := int(float64(ss.TotalAwakeTimeMilli) / total * float64(barW))
		remainder := barW - lightW - swsW2 - remW - awakeW
		lightW += remainder

		t, _ := time.Parse(time.RFC3339, s.Start)
		bar := awakeStyle.Render(strings.Repeat("█", awakeW)) +
			lightStyle.Render(strings.Repeat("█", lightW)) +
			swsStyle.Render(strings.Repeat("█", swsW2)) +
			remStyle.Render(strings.Repeat("█", remW))

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
		totalNeed := float64(need.BaselineMilli+need.NeedFromSleepDebtMilli+need.NeedFromRecentStrainMilli+need.NeedFromRecentNapMilli) / 3600000.0
		actual := float64(s.Score.StageSummary.TotalInBedTimeMilli-s.Score.StageSummary.TotalAwakeTimeMilli) / 3600000.0
		sections = append(sections, fmt.Sprintf("\n%s  Need: %.1fh  Actual: %.1fh",
			tui.TitleStyle.Render("Sleep Need vs Actual"),
			totalNeed, actual))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

type sleepSummaryStats struct {
	avgActualHours    float64
	avgGapHours       float64
	avgPerformancePct float64
	avgEfficiencyPct  float64
	avgConsistencyPct float64
	avgDisturbances   float64
}

func sleepSummary(trend []store.SleepTrendPoint, sleeps []models.Sleep) sleepSummaryStats {
	var stats sleepSummaryStats
	if len(trend) > 0 {
		var totalActual, totalPerf, totalEff, totalCons float64
		for _, t := range trend {
			totalActual += float64(t.DurationMilli) / 3600000.0
			totalPerf += t.PerformancePct
			totalEff += t.EfficiencyPct
			totalCons += t.ConsistencyPct
		}
		n := float64(len(trend))
		stats.avgActualHours = totalActual / n
		stats.avgPerformancePct = totalPerf / n
		stats.avgEfficiencyPct = totalEff / n
		stats.avgConsistencyPct = totalCons / n
	}

	var totalGap, totalDisturbances float64
	var count int
	for _, s := range sleeps {
		if s.Score == nil {
			continue
		}
		totalGap += sleepGapHours(s)
		totalDisturbances += float64(s.Score.StageSummary.DisturbanceCount)
		count++
	}
	if count > 0 {
		stats.avgGapHours = totalGap / float64(count)
		stats.avgDisturbances = totalDisturbances / float64(count)
	}
	return stats
}

func sleepNeedGaps(sleeps []models.Sleep) []float64 {
	gaps := make([]float64, 0, len(sleeps))
	for i := len(sleeps) - 1; i >= 0; i-- {
		if sleeps[i].Score == nil {
			continue
		}
		gaps = append(gaps, sleepGapHours(sleeps[i]))
	}
	return gaps
}

func recentSleepTable(sleeps []models.Sleep, limit int) string {
	if limit > len(sleeps) {
		limit = len(sleeps)
	}
	rows := make([]string, 0, limit+1)
	rows = append(rows, "  Date    Actual  Need   Gap    Perf  Eff  Cons  Dist")
	for i := 0; i < limit; i++ {
		s := sleeps[i]
		if s.Score == nil {
			continue
		}
		t, _ := time.Parse(time.RFC3339, s.Start)
		actual := actualSleepHours(s)
		need := sleepNeedHours(s)
		gap := actual - need
		score := s.Score
		rows = append(rows, fmt.Sprintf(
			"  %-6s  %5.1fh  %4.1fh  %+5.1fh  %4.0f  %3.0f  %4.0f  %4d",
			t.Format("Jan 02"),
			actual,
			need,
			gap,
			score.SleepPerformancePct,
			score.SleepEfficiencyPct,
			score.SleepConsistencyPct,
			score.StageSummary.DisturbanceCount,
		))
	}
	if len(rows) == 1 {
		return ""
	}
	return strings.Join(rows, "\n")
}

func actualSleepHours(s models.Sleep) float64 {
	if s.Score == nil {
		return 0
	}
	ss := s.Score.StageSummary
	return float64(ss.TotalInBedTimeMilli-ss.TotalAwakeTimeMilli) / 3600000.0
}

func sleepNeedHours(s models.Sleep) float64 {
	if s.Score == nil {
		return 0
	}
	need := s.Score.SleepNeeded
	return float64(need.BaselineMilli+need.NeedFromSleepDebtMilli+need.NeedFromRecentStrainMilli+need.NeedFromRecentNapMilli) / 3600000.0
}

func sleepGapHours(s models.Sleep) float64 {
	return actualSleepHours(s) - sleepNeedHours(s)
}

func (m *SleepModel) sparkWidth() int {
	w := m.width - 10
	if w < 20 {
		w = 20
	}
	if w > 60 {
		w = 60
	}
	return w
}
