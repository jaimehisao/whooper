package views

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"git.infra.hisao.org/hisao/whooper/internal/analysis"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	"git.infra.hisao.org/hisao/whooper/internal/tui"
	"git.infra.hisao.org/hisao/whooper/internal/tui/components"
)

var recoveryRanges = []int{7, 14, 30, 90}

type RecoveryModel struct {
	db       *store.DB
	width    int
	rangeIdx int
	data     []store.RecoveryTrendPoint
	loaded   bool
	err      string
}

func NewRecovery(db *store.DB) RecoveryModel {
	return RecoveryModel{db: db, rangeIdx: 2, width: 80}
}

type recoveryDataMsg struct {
	data []store.RecoveryTrendPoint
	err  error
}

func (m *RecoveryModel) Init() tea.Cmd {
	return m.Refresh()
}

func (m *RecoveryModel) Refresh() tea.Cmd {
	db := m.db
	rangeIdx := m.rangeIdx
	return func() tea.Msg {
		days := recoveryRanges[rangeIdx]
		from := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour).Format("2006-01-02")
		data, err := db.GetRecoveryTrend(from, "")
		return recoveryDataMsg{data: data, err: err}
	}
}

func (m *RecoveryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case recoveryDataMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
		}
		m.data = msg.data
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
			if m.rangeIdx < len(recoveryRanges)-1 {
				m.rangeIdx++
				return m, m.Refresh()
			}
		}
	}
	return m, nil
}

func (m *RecoveryModel) View() string {
	if !m.loaded {
		return tui.MutedStyle.Render("Loading recovery data...")
	}

	days := recoveryRanges[m.rangeIdx]
	header := tui.TitleStyle.Render(fmt.Sprintf("Recovery Trends (%dd)  < %dd >", days, days))
	sparkW := m.sparkWidth()

	sections := make([]string, 0, 8)
	sections = append(sections, header)

	if m.err != "" {
		sections = append(sections, tui.RedStyle.Render("Error: "+m.err))
	}

	if len(m.data) == 0 {
		sections = append(sections, tui.MutedStyle.Render("No recovery data available. Run 'whooper sync' first."))
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}

	recScores := make([]float64, 0, len(m.data))
	hrvValues := make([]float64, 0, len(m.data))
	rhrValues := make([]float64, 0, len(m.data))
	for _, d := range m.data {
		recScores = append(recScores, d.RecoveryScore)
		hrvValues = append(hrvValues, d.HRV)
		rhrValues = append(rhrValues, d.RHR)
	}

	recMA := analysis.MovingAverage(recScores, 7)
	hrvMA := analysis.MovingAverage(hrvValues, 7)

	last := m.data[len(m.data)-1]
	current := fmt.Sprintf("Latest: %s  %s  %s",
		tui.RecoveryColor(last.RecoveryScore).Render(fmt.Sprintf("Recovery: %.0f%%", last.RecoveryScore)),
		tui.AccentStyle.Render(fmt.Sprintf("HRV: %.0f ms", last.HRV)),
		tui.AccentStyle.Render(fmt.Sprintf("RHR: %.0f bpm", last.RHR)))
	sections = append(sections, current)

	recSpark := components.Sparkline(recScores, sparkW)
	sections = append(sections, fmt.Sprintf("\n%s\n%s",
		tui.TextStyle.Render("Recovery Score"),
		tui.GreenStyle.Render(recSpark)))

	if len(recMA) > 0 {
		maSpark := components.Sparkline(recMA, sparkW)
		sections = append(sections, fmt.Sprintf("%s %s",
			tui.MutedStyle.Render("7d MA:"),
			tui.MutedStyle.Render(maSpark)))
	}

	hrvSpark := components.Sparkline(hrvValues, sparkW)
	sections = append(sections, fmt.Sprintf("\n%s\n%s",
		tui.TextStyle.Render("HRV (ms)"),
		tui.AccentStyle.Render(hrvSpark)))

	if len(hrvMA) > 0 {
		maSpark := components.Sparkline(hrvMA, sparkW)
		sections = append(sections, fmt.Sprintf("%s %s",
			tui.MutedStyle.Render("7d MA:"),
			tui.MutedStyle.Render(maSpark)))
	}

	rhrSpark := components.Sparkline(rhrValues, sparkW)
	sections = append(sections, fmt.Sprintf("\n%s\n%s",
		tui.TextStyle.Render("Resting Heart Rate (bpm)"),
		tui.RedStyle.Render(rhrSpark)))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m *RecoveryModel) sparkWidth() int {
	w := m.width - 10
	if w < 20 {
		w = 20
	}
	if w > 60 {
		w = 60
	}
	return w
}
