package views

import (
	"fmt"
	"sort"
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
	height   int
	rangeIdx int
	workouts []models.Workout
	cursor   int
	detail   bool
	loaded   bool
	err      string
}

func NewWorkouts(db *store.DB) WorkoutsModel {
	return WorkoutsModel{db: db, rangeIdx: 2, width: 80, height: 24}
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
		m.height = msg.Height
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

	sections := make([]string, 0, 4)
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

	summary := workoutSummary(m.workouts)
	sections = append(sections, fmt.Sprintf(
		"\nWorkouts: %d  Total strain: %.1f  Avg strain: %.1f  Time: %s  Distance: %.1f km  Avg HR: %.0f",
		summary.count,
		summary.totalStrain,
		summary.avgStrain,
		formatMinutes(summary.totalMinutes),
		summary.totalDistanceKm,
		summary.avgHR,
	))

	strains := workoutStrainsOldestFirst(m.workouts)
	if len(strains) > 0 {
		sections = append(sections, fmt.Sprintf("\n%s\n%s",
			tui.TextStyle.Render("Workout Strain"),
			tui.AccentStyle.Render(components.Sparkline(strains, m.sparkWidth()))))
	}

	breakdown := sportBreakdown(m.workouts, 6)
	if breakdown != "" {
		sections = append(sections, "\n"+tui.TitleStyle.Render("Sport Breakdown"))
		sections = append(sections, breakdown)
	}

	// Virtualization: calculate viewport
	// Reserved height for header, footer, table headers etc.
	reservedHeight := 16
	if m.err != "" {
		reservedHeight++
	}
	viewHeight := m.height - reservedHeight
	if viewHeight < 1 {
		viewHeight = 1
	}

	startIdx := 0
	if m.cursor >= viewHeight {
		startIdx = m.cursor - viewHeight + 1
	}
	endIdx := startIdx + viewHeight
	if endIdx > len(m.workouts) {
		endIdx = len(m.workouts)
		startIdx = endIdx - viewHeight
		if startIdx < 0 {
			startIdx = 0
		}
	}

	// Responsive column widths
	sportW := 20
	if m.width < 90 {
		sportW = 14
	}
	headers := []string{"Date", "Sport", "Strain", "Avg HR", "Max HR", "Duration"}
	widths := []int{12, sportW, 8, 8, 8, 10}

	rows := make([][]string, 0, endIdx-startIdx)
	for i := startIdx; i < endIdx; i++ {
		w := m.workouts[i]
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

	table := components.HighlightedTable(headers, rows, widths, m.cursor-startIdx)
	sections = append(sections, table)
	sections = append(sections, tui.MutedStyle.Render(fmt.Sprintf("  ↑↓ navigate (%d/%d)  enter detail  esc back", m.cursor+1, len(m.workouts))))
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m *WorkoutsModel) detailView() string {
	w := m.workouts[m.cursor]
	sport := models.SportName[w.SportID]
	if sport == "" {
		sport = fmt.Sprintf("Sport %d", w.SportID)
	}

	t, _ := time.Parse(time.RFC3339, w.Start)
	lines := make([]string, 0, 16)
	lines = append(lines, tui.TitleStyle.Render(fmt.Sprintf("Workout Detail: %s", sport)))
	lines = append(lines, fmt.Sprintf("  Date:     %s", t.Format("2006-01-02 15:04")))
	lines = append(lines, fmt.Sprintf("  Duration: %s", calcDuration(w.Start, w.End)))

	if w.Score != nil {
		s := w.Score
		lines = append(lines, fmt.Sprintf("  Strain:   %.1f", s.Strain))
		lines = append(lines, fmt.Sprintf("  Avg HR:   %d bpm", s.AverageHeartRate))
		lines = append(lines, fmt.Sprintf("  Max HR:   %d bpm", s.MaxHeartRate))
		lines = append(lines, fmt.Sprintf("  Calories: %.0f kJ", s.Kilojoule))
		lines = append(lines, fmt.Sprintf("  Recorded: %.0f%%", s.PercentRecorded))
		hours := workoutDurationHours(w)
		if hours > 0 {
			lines = append(lines, fmt.Sprintf("  Load:     %.1f strain/hour", s.Strain/hours))
		}
		if s.DistanceOrZero() > 0 {
			lines = append(lines, fmt.Sprintf("  Distance: %.1f km", s.DistanceOrZero()/1000))
		}
		if s.AltitudeGainOrZero() > 0 {
			lines = append(lines, fmt.Sprintf("  Elev Gain: %.0f m", s.AltitudeGainOrZero()))
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

type workoutSummaryStats struct {
	count           int
	totalStrain     float64
	avgStrain       float64
	totalMinutes    float64
	totalDistanceKm float64
	avgHR           float64
}

type sportSummary struct {
	name       string
	count      int
	strain     float64
	minutes    float64
	distanceKm float64
}

func workoutSummary(workouts []models.Workout) workoutSummaryStats {
	stats := workoutSummaryStats{count: len(workouts)}
	var strainCount, hrCount int
	var hrTotal float64
	for _, w := range workouts {
		stats.totalMinutes += workoutDurationMinutes(w)
		if w.Score == nil {
			continue
		}
		stats.totalStrain += w.Score.Strain
		strainCount++
		if w.Score.DistanceOrZero() > 0 {
			stats.totalDistanceKm += w.Score.DistanceOrZero() / 1000
		}
		if w.Score.AverageHeartRate > 0 {
			hrTotal += float64(w.Score.AverageHeartRate)
			hrCount++
		}
	}
	if strainCount > 0 {
		stats.avgStrain = stats.totalStrain / float64(strainCount)
	}
	if hrCount > 0 {
		stats.avgHR = hrTotal / float64(hrCount)
	}
	return stats
}

func workoutStrainsOldestFirst(workouts []models.Workout) []float64 {
	strains := make([]float64, 0, len(workouts))
	for i := len(workouts) - 1; i >= 0; i-- {
		if workouts[i].Score == nil {
			continue
		}
		strains = append(strains, workouts[i].Score.Strain)
	}
	return strains
}

func sportBreakdown(workouts []models.Workout, limit int) string {
	bySport := make(map[int]*sportSummary)
	for _, w := range workouts {
		summary := bySport[w.SportID]
		if summary == nil {
			summary = &sportSummary{name: sportName(w.SportID)}
			bySport[w.SportID] = summary
		}
		summary.count++
		summary.minutes += workoutDurationMinutes(w)
		if w.Score != nil {
			summary.strain += w.Score.Strain
			if w.Score.DistanceOrZero() > 0 {
				summary.distanceKm += w.Score.DistanceOrZero() / 1000
			}
		}
	}

	sports := make([]sportSummary, 0, len(bySport))
	for _, summary := range bySport {
		sports = append(sports, *summary)
	}
	sort.Slice(sports, func(i, j int) bool {
		if sports[i].strain == sports[j].strain {
			return sports[i].count > sports[j].count
		}
		return sports[i].strain > sports[j].strain
	})
	if limit > len(sports) {
		limit = len(sports)
	}

	lines := make([]string, 0, limit+1)
	lines = append(lines, "  Sport           Cnt  Strain  Time    Distance")
	for i := 0; i < limit; i++ {
		s := sports[i]
		lines = append(lines, fmt.Sprintf("  %-14s  %3d  %6.1f  %-6s  %6.1f km",
			s.name,
			s.count,
			s.strain,
			formatMinutes(s.minutes),
			s.distanceKm,
		))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func sportName(id int) string {
	sport := models.SportName[id]
	if sport == "" {
		return fmt.Sprintf("Sport %d", id)
	}
	return sport
}

func workoutDurationMinutes(w models.Workout) float64 {
	s, err1 := time.Parse(time.RFC3339, w.Start)
	e, err2 := time.Parse(time.RFC3339, w.End)
	if err1 != nil || err2 != nil {
		return 0
	}
	return e.Sub(s).Minutes()
}

func workoutDurationHours(w models.Workout) float64 {
	return workoutDurationMinutes(w) / 60
}

func formatMinutes(minutes float64) string {
	if minutes < 0 {
		minutes = 0
	}
	total := int(minutes + 0.5)
	hours := total / 60
	mins := total % 60
	return fmt.Sprintf("%dh%02dm", hours, mins)
}

func (m *WorkoutsModel) sparkWidth() int {
	w := m.width - 10
	if w < 20 {
		w = 20
	}
	if w > 60 {
		w = 60
	}
	return w
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
