package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"git.infra.hisao.org/hisao/whooper/internal/analysis"
	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/models"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	"git.infra.hisao.org/hisao/whooper/internal/tui"
	"git.infra.hisao.org/hisao/whooper/internal/tui/components"
)

type DashboardModel struct {
	db             *store.DB
	cfg            *config.Config
	canSync        bool
	width          int
	height         int
	recoveryScore  float64
	hrvValue       float64
	rhrValue       float64
	sleepHours     float64
	sleepEffPct    float64
	dayStrain      float64
	recoveryDate   string
	sleepDate      string
	strainDate     string
	sparklineData  []float64
	recentWorkouts []models.Workout
	alerts         []string
	lastSync       time.Time
	isStale        bool
	loaded         bool
	err            string
}

func NewDashboard(db *store.DB) DashboardModel {
	return DashboardModel{db: db, width: 80, canSync: true}
}

// SetConfig attaches alert configuration used by the dashboard.
func (m *DashboardModel) SetConfig(cfg *config.Config) {
	m.cfg = cfg
}

// SetCanSync controls whether the dashboard prompts the user to press 's'.
func (m *DashboardModel) SetCanSync(can bool) {
	m.canSync = can
}

type dashboardDataMsg struct {
	recovery   float64
	hrv        float64
	rhr        float64
	sleepHrs   float64
	sleepEff   float64
	strain     float64
	recDate    string
	sleepDate  string
	strainDate string
	sparkline  []float64
	workouts   []models.Workout
	alerts     []string
	lastSync   time.Time
	err        string
}

func (m *DashboardModel) Init() tea.Cmd {
	return m.Refresh()
}

func (m *DashboardModel) Refresh() tea.Cmd {
	db := m.db
	cfg := m.cfg
	return func() tea.Msg {
		now := time.Now().UTC()
		weekAgo := now.Add(-7 * 24 * time.Hour).Format("2006-01-02")
		monthAgo := now.Add(-30 * 24 * time.Hour).Format("2006-01-02")
		today := now.Format("2006-01-02")

		msg := dashboardDataMsg{}
		var errs []string

		lastSyncStr, err := db.GetSyncState("cycles")
		if err == nil && lastSyncStr != "" {
			msg.lastSync, _ = time.Parse(time.RFC3339, lastSyncStr)
		}

		recoveries, err := db.GetRecoveryTrend(monthAgo, today)
		if err != nil {
			errs = append(errs, fmt.Sprintf("recovery: %v", err))
		} else if len(recoveries) > 0 {
			latest := recoveries[len(recoveries)-1]
			msg.recovery = latest.RecoveryScore
			msg.hrv = latest.HRV
			msg.rhr = latest.RHR
			msg.recDate = latest.Date
		}

		sleeps, err := db.GetSleepTrend(monthAgo, today)
		if err != nil {
			errs = append(errs, fmt.Sprintf("sleep: %v", err))
		} else if len(sleeps) > 0 {
			latest := sleeps[len(sleeps)-1]
			msg.sleepHrs = float64(latest.DurationMilli) / 3600000.0
			msg.sleepEff = latest.EfficiencyPct
			msg.sleepDate = latest.Date
		}

		strains, err := db.GetStrainTrend(monthAgo, today)
		if err != nil {
			errs = append(errs, fmt.Sprintf("strain: %v", err))
		} else if len(strains) > 0 {
			latest := strains[len(strains)-1]
			msg.strain = latest.Strain
			msg.strainDate = latest.Date
		}

		weekRecoveries, err := db.GetRecoveryTrend(weekAgo, today)
		if err != nil {
			errs = append(errs, fmt.Sprintf("week recovery: %v", err))
		}
		msg.sparkline = make([]float64, 0, len(weekRecoveries))
		for _, r := range weekRecoveries {
			msg.sparkline = append(msg.sparkline, r.RecoveryScore)
		}

		workouts, err := db.ListWorkouts(weekAgo, today)
		if err != nil {
			errs = append(errs, fmt.Sprintf("workouts: %v", err))
		}
		if len(workouts) > 5 {
			workouts = workouts[:5]
		}
		msg.workouts = workouts

		if cfg != nil {
			alerts, alertErr := analysis.CheckAlerts(db, cfg)
			if alertErr != nil {
				errs = append(errs, fmt.Sprintf("alerts: %v", alertErr))
			} else {
				for _, a := range alerts {
					msg.alerts = append(msg.alerts, a.Message)
				}
			}
		}

		if len(errs) > 0 {
			msg.err = strings.Join(errs, "; ")
		}
		return msg
	}
}

func (m *DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dashboardDataMsg:
		m.recoveryScore = msg.recovery
		m.hrvValue = msg.hrv
		m.rhrValue = msg.rhr
		m.sleepHours = msg.sleepHrs
		m.sleepEffPct = msg.sleepEff
		m.dayStrain = msg.strain
		m.recoveryDate = msg.recDate
		m.sleepDate = msg.sleepDate
		m.strainDate = msg.strainDate
		m.sparklineData = msg.sparkline
		m.recentWorkouts = msg.workouts
		m.alerts = msg.alerts
		m.lastSync = msg.lastSync
		m.isStale = !m.lastSync.IsZero() && time.Since(m.lastSync) > 24*time.Hour
		m.err = msg.err
		m.loaded = true
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m *DashboardModel) View() string {
	if !m.loaded {
		return tui.MutedStyle.Render("Loading dashboard...")
	}

	sparkW := m.sparkWidth()
	sections := make([]string, 0, 10+len(m.alerts))

	statusLine := ""
	if m.lastSync.IsZero() {
		if m.canSync {
			statusLine = tui.YellowStyle.Render("Never synced. Press 's' to sync.")
		} else {
			statusLine = tui.YellowStyle.Render("Never synced. Run 'whooper login' then sync.")
		}
	} else {
		since := time.Since(m.lastSync).Round(time.Minute)
		status := fmt.Sprintf("Last sync: %s ago", since)
		if m.isStale {
			statusLine = tui.YellowStyle.Render("! " + status + " (stale)")
		} else {
			statusLine = tui.MutedStyle.Render(status)
		}
	}
	sections = append(sections, statusLine)

	if m.err != "" {
		sections = append(sections, tui.RedStyle.Render("Error: "+m.err))
	}

	for _, alert := range m.alerts {
		sections = append(sections, tui.YellowStyle.Render("  ! "+alert))
	}

	hasData := m.recoveryDate != "" || m.sleepDate != "" || m.strainDate != "" ||
		m.recoveryScore > 0 || m.hrvValue > 0 || m.rhrValue > 0 ||
		m.sleepHours > 0 || m.dayStrain > 0 || len(m.alerts) > 0 ||
		len(m.recentWorkouts) > 0 || len(m.sparklineData) > 0

	if !hasData {
		sections = append(sections, "", tui.MutedStyle.Render("No local data found. Run 'whooper sync' to fetch your data."))
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}

	todayPanel := m.renderTodayPanel(sparkW)
	sections = append(sections, "", todayPanel)

	dateLine := latestDateLine(m.recoveryDate, m.sleepDate, m.strainDate)
	if dateLine != "" {
		sections = append(sections, tui.MutedStyle.Render(dateLine))
	}

	if len(m.sparklineData) > 0 {
		spark := components.Sparkline(m.sparklineData, sparkW)
		sparkSection := fmt.Sprintf("\n%s\n%s",
			tui.TitleStyle.Render("7-Day Recovery"),
			tui.GreenStyle.Render(spark))
		sections = append(sections, sparkSection)
	}

	if len(m.recentWorkouts) > 0 {
		woLines := make([]string, 0, len(m.recentWorkouts)+1)
		woLines = append(woLines, "\n"+tui.TitleStyle.Render("Recent Workouts"))
		for _, w := range m.recentWorkouts {
			sport := models.SportName[w.SportID]
			if sport == "" {
				sport = fmt.Sprintf("Sport %d", w.SportID)
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

func (m *DashboardModel) renderTodayPanel(width int) string {
	gauge := components.Gauge(m.recoveryScore, width)

	metrics := fmt.Sprintf(
		"%s  %s  %s  %s",
		tui.RecoveryColor(m.recoveryScore).Render(fmt.Sprintf("HRV: %.0f ms", m.hrvValue)),
		tui.AccentStyle.Render(fmt.Sprintf("RHR: %.0f bpm", m.rhrValue)),
		tui.AccentStyle.Render(fmt.Sprintf("Sleep: %.1fh (%.0f%%)", m.sleepHours, m.sleepEffPct)),
		tui.AccentStyle.Render(fmt.Sprintf("Strain: %.1f", m.dayStrain)),
	)

	return tui.BoxStyle.Width(width + 4).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			tui.TitleStyle.Render("Current Status"),
			gauge,
			"",
			metrics,
		),
	)
}

func latestDateLine(recoveryDate, sleepDate, strainDate string) string {
	parts := make([]string, 0, 3)
	if recoveryDate != "" {
		parts = append(parts, "recovery "+formatDashboardDate(recoveryDate))
	}
	if sleepDate != "" {
		parts = append(parts, "sleep "+formatDashboardDate(sleepDate))
	}
	if strainDate != "" {
		parts = append(parts, "strain "+formatDashboardDate(strainDate))
	}
	if len(parts) == 0 {
		return ""
	}
	return "Latest scored: " + strings.Join(parts, " | ")
}

func formatDashboardDate(raw string) string {
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return raw
	}
	return t.Format("Jan 02")
}

func (m *DashboardModel) sparkWidth() int {
	w := m.width - 10
	if w < 20 {
		w = 20
	}
	if w > 60 {
		w = 60
	}
	return w
}
