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

type DashboardModel struct {
	db             *store.DB
	recoveryScore  float64
	hrvValue       float64
	rhrValue       float64
	sleepHours     float64
	sleepEffPct    float64
	dayStrain      float64
	sparklineData  []float64
	recentWorkouts []models.Workout
	loaded         bool
}

func NewDashboard(db *store.DB) DashboardModel {
	return DashboardModel{db: db}
}

type dashboardDataMsg struct {
	recovery      float64
	hrv           float64
	rhr           float64
	sleepHours    float64
	sleepEff      float64
	strain        float64
	sparkline     []float64
	workouts      []models.Workout
}

func (m *DashboardModel) Init() tea.Cmd {
	return m.Refresh()
}

func (m *DashboardModel) Refresh() tea.Cmd {
	db := m.db
	return func() tea.Msg {
		now := time.Now().UTC()
		today := now.Format("2006-01-02")
		weekAgo := now.Add(-7 * 24 * time.Hour).Format("2006-01-02")

		msg := dashboardDataMsg{}

		recoveries, _ := db.GetRecoveryTrend(today, "")
		if len(recoveries) > 0 {
			msg.recovery = recoveries[0].RecoveryScore
			msg.hrv = recoveries[0].HRV
			msg.rhr = recoveries[0].RHR
		}

		sleeps, _ := db.GetSleepTrend(today, "")
		if len(sleeps) > 0 {
			msg.sleepHours = float64(sleeps[0].DurationMilli) / 3600000.0
			msg.sleepEff = sleeps[0].EfficiencyPct
		}

		strains, _ := db.GetStrainTrend(today, "")
		if len(strains) > 0 {
			msg.strain = strains[0].Strain
		}

		weekRecoveries, _ := db.GetRecoveryTrend(weekAgo, "")
		for _, r := range weekRecoveries {
			msg.sparkline = append(msg.sparkline, r.RecoveryScore)
		}

		workouts, _ := db.ListWorkouts(weekAgo, "")
		if len(workouts) > 5 {
			workouts = workouts[:5]
		}
		msg.workouts = workouts

		return msg
	}
}

func (m *DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dashboardDataMsg:
		m.recoveryScore = msg.recovery
		m.hrvValue = msg.hrv
		m.rhrValue = msg.rhr
		m.sleepHours = msg.sleepHours
		m.sleepEffPct = msg.sleepEff
		m.dayStrain = msg.strain
		m.sparklineData = msg.sparkline
		m.recentWorkouts = msg.workouts
		m.loaded = true
	}
	return m, nil
}

func (m *DashboardModel) View() string {
	if !m.loaded {
		return tui.MutedStyle.Render("Loading dashboard...")
	}

	var sections []string

	gauge := components.Gauge(m.recoveryScore, 30)
	sections = append(sections, tui.BoxStyle.Render(gauge))

	metrics := fmt.Sprintf(
		"%s  %s  %s  %s",
		tui.RecoveryColor(m.recoveryScore).Render(fmt.Sprintf("HRV: %.0f ms", m.hrvValue)),
		tui.AccentStyle.Render(fmt.Sprintf("RHR: %.0f bpm", m.rhrValue)),
		tui.AccentStyle.Render(fmt.Sprintf("Sleep: %.1fh (%.0f%%)", m.sleepHours, m.sleepEffPct)),
		tui.AccentStyle.Render(fmt.Sprintf("Strain: %.1f", m.dayStrain)),
	)
	sections = append(sections, metrics)

	if len(m.sparklineData) > 0 {
		spark := components.Sparkline(m.sparklineData, 30)
		sparkSection := fmt.Sprintf("%s\n%s",
			tui.TitleStyle.Render("7-Day Recovery"),
			tui.GreenStyle.Render(spark))
		sections = append(sections, sparkSection)
	}

	if len(m.recentWorkouts) > 0 {
		var woLines []string
		woLines = append(woLines, tui.TitleStyle.Render("Recent Workouts"))
		for _, w := range m.recentWorkouts {
			sport := models.SportName[w.SportID]
			if sport == "" {
				sport = "Unknown"
			}
			strain := 0.0
			if w.Score != nil {
				strain = w.Score.Strain
			}
			t, _ := time.Parse(time.RFC3339, w.Start)
			woLines = append(woLines, fmt.Sprintf("  %s  %-20s  strain: %.1f",
				t.Format("Jan 02"), sport, strain))
		}
		sections = append(sections, strings.Join(woLines, "\n"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
