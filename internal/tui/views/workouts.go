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

var workoutRanges = []int{7, 14, 30, 90}

type WorkoutsModel struct {
	db       *store.DB
	width    int
	rangeIdx int
	workouts []models.Workout
	cursor   int
	detail   bool
	loaded   bool
	err      string
}

func NewWorkouts(db *store.DB) WorkoutsModel {
	return WorkoutsModel{db: db, rangeIdx: 2, width: 80}
}

type workoutsDataMsg struct {
	workouts []models.Workout
	err      error
}

func (m *WorkoutsModel) Init() tea.Cmd {
	return m.Refresh()
}

func (m *WorkoutsModel) Refresh() tea.Cmd {
	db := m.db
	rangeIdx := m.rangeIdx
	return func() tea.Msg {
		days := workoutRanges[rangeIdx]
		from := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour).Format("2006-01-02")
		workouts, err := db.ListWorkouts(from, "")
		return workoutsDataMsg{workouts: workouts, err: err}
	}
}

func (m *WorkoutsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case workoutsDataMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
		}
		m.workouts = msg.workouts
		m.loaded = true
		m.cursor = 0
		m.detail = false
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.workouts)-1 {
				m.cursor++
			}
		case "enter":
			m.detail = !m.detail
		case "left", "<":
			if m.rangeIdx > 0 {
				m.rangeIdx--
				return m, m.Refresh()
			}
		case "right", ">":
			if m.rangeIdx < len(workoutRanges)-1 {
				m.rangeIdx++
				return m, m.Refresh()
			}
		case "esc":
			m.detail = false
		}
	}
	return m, nil
}

func (m *WorkoutsModel) View() string {
	if !m.loaded {
		return tui.MutedStyle.Render("Loading workouts...")
	}

	days := workoutRanges[m.rangeIdx]
	header := tui.TitleStyle.Render(fmt.Sprintf("Workouts (%dd)  < %dd >", days, days))

	var sections []string
	sections = append(sections, header)

	if m.err != "" {
		sections = append(sections, tui.RedStyle.Render("Error: "+m.err))
	}

	if len(m.workouts) == 0 {
		sections = append(sections, tui.MutedStyle.Render("No workouts found. Run 'whooper sync' first."))
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}

	if m.detail && m.cursor < len(m.workouts) {
		return m.detailView()
	}

	// Responsive column widths
	sportW := 20
	if m.width < 90 {
		sportW = 14
	}
	headers := []string{"Date", "Sport", "Strain", "Avg HR", "Max HR", "Duration"}
	widths := []int{12, sportW, 8, 8, 8, 10}

	var rows [][]string
	for _, w := range m.workouts {
		sport := models.SportName[w.SportID]
		if sport == "" {
			sport = fmt.Sprintf("Sport %d", w.SportID)
		}
		strain, avgHR, maxHR := "", "", ""
		if w.Score != nil {
			strain = fmt.Sprintf("%.1f", w.Score.Strain)
			avgHR = fmt.Sprintf("%d", w.Score.AverageHeartRate)
			maxHR = fmt.Sprintf("%d", w.Score.MaxHeartRate)
		}
		duration := calcDuration(w.Start, w.End)
		t, _ := time.Parse(time.RFC3339, w.Start)
		rows = append(rows, []string{
			t.Format("Jan 02 15:04"), sport, strain, avgHR, maxHR, duration,
		})
	}

	table := components.HighlightedTable(headers, rows, widths, m.cursor)
	sections = append(sections, table)
	sections = append(sections, tui.MutedStyle.Render("  ↑↓ navigate  enter detail  esc back"))
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m *WorkoutsModel) detailView() string {
	w := m.workouts[m.cursor]
	sport := models.SportName[w.SportID]
	if sport == "" {
		sport = fmt.Sprintf("Sport %d", w.SportID)
	}

	t, _ := time.Parse(time.RFC3339, w.Start)
	var lines []string
	lines = append(lines, tui.TitleStyle.Render(fmt.Sprintf("Workout Detail: %s", sport)))
	lines = append(lines, fmt.Sprintf("  Date:     %s", t.Format("2006-01-02 15:04")))
	lines = append(lines, fmt.Sprintf("  Duration: %s", calcDuration(w.Start, w.End)))

	if w.Score != nil {
		s := w.Score
		lines = append(lines, fmt.Sprintf("  Strain:   %.1f", s.Strain))
		lines = append(lines, fmt.Sprintf("  Avg HR:   %d bpm", s.AverageHeartRate))
		lines = append(lines, fmt.Sprintf("  Max HR:   %d bpm", s.MaxHeartRate))
		lines = append(lines, fmt.Sprintf("  Calories: %.0f kJ", s.Kilojoule))
		if s.DistanceMeter > 0 {
			lines = append(lines, fmt.Sprintf("  Distance: %.1f km", s.DistanceMeter/1000))
		}
		if s.ZoneDuration != nil {
			zd := s.ZoneDuration
			total := float64(zd.ZoneZeroMilli+zd.ZoneOneMilli+zd.ZoneTwoMilli+zd.ZoneThreeMilli+zd.ZoneFourMilli+zd.ZoneFiveMilli) / 60000
			lines = append(lines, "", tui.AccentStyle.Render("  HR Zones:"))
			items := []components.BarItem{
				{Label: "Zone 0", Value: float64(zd.ZoneZeroMilli) / 60000, MaxValue: total, Color: lipgloss.Color("#666666")},
				{Label: "Zone 1", Value: float64(zd.ZoneOneMilli) / 60000, MaxValue: total, Color: lipgloss.Color("#5B9BD5")},
				{Label: "Zone 2", Value: float64(zd.ZoneTwoMilli) / 60000, MaxValue: total, Color: lipgloss.Color("#00D46A")},
				{Label: "Zone 3", Value: float64(zd.ZoneThreeMilli) / 60000, MaxValue: total, Color: lipgloss.Color("#FFCC00")},
				{Label: "Zone 4", Value: float64(zd.ZoneFourMilli) / 60000, MaxValue: total, Color: lipgloss.Color("#FF6600")},
				{Label: "Zone 5", Value: float64(zd.ZoneFiveMilli) / 60000, MaxValue: total, Color: lipgloss.Color("#FF3333")},
			}
			barW := 25
			if m.width > 100 {
				barW = 35
			}
			lines = append(lines, components.BarChart(items, barW))
		}
	}

	lines = append(lines, "\n"+tui.MutedStyle.Render("  esc/enter to go back"))
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func calcDuration(start, end string) string {
	s, err1 := time.Parse(time.RFC3339, start)
	e, err2 := time.Parse(time.RFC3339, end)
	if err1 != nil || err2 != nil {
		return "-"
	}
	d := e.Sub(s)
	h := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%02dm", h, mins)
}
