package views

import (
	"fmt"
	"math"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"git.infra.hisao.org/hisao/whooper/internal/analysis"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	"git.infra.hisao.org/hisao/whooper/internal/tui"
)

var metricNames = []string{"recovery", "hrv", "rhr", "strain", "sleep_duration", "sleep_efficiency"}
var metricLabels = map[string]string{
	"recovery":         "Recovery %",
	"hrv":              "HRV (ms)",
	"rhr":              "RHR (bpm)",
	"strain":           "Strain",
	"sleep_duration":   "Sleep Duration",
	"sleep_efficiency": "Sleep Efficiency %",
}

type CorrelationsModel struct {
	db     *store.DB
	width  int
	xIdx   int
	yIdx   int
	data   []store.CorrelationPoint
	r      float64
	loaded bool
	err    string
}

func NewCorrelations(db *store.DB) CorrelationsModel {
	return CorrelationsModel{db: db, xIdx: 1, yIdx: 0, width: 80}
}

type correlationDataMsg struct {
	data []store.CorrelationPoint
	r    float64
	err  error
}

func (m *CorrelationsModel) Init() tea.Cmd {
	return m.Refresh()
}

func (m *CorrelationsModel) Refresh() tea.Cmd {
	db := m.db
	xIdx, yIdx := m.xIdx, m.yIdx
	return func() tea.Msg {
		xMetric := metricNames[xIdx]
		yMetric := metricNames[yIdx]
		data, err := db.GetCorrelationData(xMetric, yMetric)
		if err != nil {
			return correlationDataMsg{err: err}
		}

		xs := make([]float64, 0, len(data))
		ys := make([]float64, 0, len(data))
		for _, p := range data {
			xs = append(xs, p.X)
			ys = append(ys, p.Y)
		}
		r := analysis.PearsonCorrelation(xs, ys)
		return correlationDataMsg{data: data, r: r}
	}
}

func (m *CorrelationsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case correlationDataMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
		}
		m.data = msg.data
		m.r = msg.r
		m.loaded = true
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "<":
			m.xIdx = (m.xIdx + len(metricNames) - 1) % len(metricNames)
			if m.xIdx == m.yIdx {
				m.xIdx = (m.xIdx + len(metricNames) - 1) % len(metricNames)
			}
			return m, m.Refresh()
		case "right", ">":
			m.xIdx = (m.xIdx + 1) % len(metricNames)
			if m.xIdx == m.yIdx {
				m.xIdx = (m.xIdx + 1) % len(metricNames)
			}
			return m, m.Refresh()
		case "[":
			m.yIdx = (m.yIdx + len(metricNames) - 1) % len(metricNames)
			if m.yIdx == m.xIdx {
				m.yIdx = (m.yIdx + len(metricNames) - 1) % len(metricNames)
			}
			return m, m.Refresh()
		case "]":
			m.yIdx = (m.yIdx + 1) % len(metricNames)
			if m.yIdx == m.xIdx {
				m.yIdx = (m.yIdx + 1) % len(metricNames)
			}
			return m, m.Refresh()
		}
	}
	return m, nil
}

func (m *CorrelationsModel) View() string {
	if !m.loaded {
		return tui.MutedStyle.Render("Loading correlation data...")
	}

	xLabel := metricLabels[metricNames[m.xIdx]]
	yLabel := metricLabels[metricNames[m.yIdx]]

	header := tui.TitleStyle.Render(fmt.Sprintf("Correlations: %s vs %s", xLabel, yLabel))

	sections := make([]string, 0, 10)
	sections = append(sections, header)

	if m.err != "" {
		sections = append(sections, tui.RedStyle.Render("Error: "+m.err))
	}

	if len(m.data) < 3 {
		sections = append(sections, tui.MutedStyle.Render("Not enough data for correlation (need 3+ points). Run 'whooper sync' first."))
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}

	rStr := fmt.Sprintf("r = %.3f", m.r)
	var rStyled string
	if math.Abs(m.r) >= 0.7 {
		rStyled = tui.GreenStyle.Render(rStr + " (strong)")
	} else if math.Abs(m.r) >= 0.4 {
		rStyled = tui.YellowStyle.Render(rStr + " (moderate)")
	} else {
		rStyled = tui.MutedStyle.Render(rStr + " (weak)")
	}

	// Responsive plot size
	plotWidth := m.width - 10
	if plotWidth < 30 {
		plotWidth = 30
	}
	if plotWidth > 60 {
		plotWidth = 60
	}
	plotHeight := 20
	if m.width < 60 {
		plotHeight = 12
	}

	scatter := renderScatter(m.data, plotWidth, plotHeight)

	sections = append(sections, fmt.Sprintf("  Pearson %s  (n=%d)", rStyled, len(m.data)))
	sections = append(sections, "")
	sections = append(sections, fmt.Sprintf("  %s (Y: %s)", tui.AccentStyle.Render("▲"), yLabel))
	sections = append(sections, scatter)
	sections = append(sections, fmt.Sprintf("  %s %s (X: %s)", strings.Repeat("─", plotWidth), tui.AccentStyle.Render("▶"), xLabel))
	sections = append(sections, "")
	sections = append(sections, tui.MutedStyle.Render("  </>: change X metric  [/]: change Y metric"))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func renderScatter(data []store.CorrelationPoint, width, height int) string {
	if len(data) == 0 {
		return ""
	}

	minX, maxX := data[0].X, data[0].X
	minY, maxY := data[0].Y, data[0].Y
	for _, p := range data {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	rangeX := maxX - minX
	rangeY := maxY - minY
	if rangeX == 0 {
		rangeX = 1
	}
	if rangeY == 0 {
		rangeY = 1
	}

	grid := make([][]int, height)
	for i := range grid {
		grid[i] = make([]int, width)
	}

	for _, p := range data {
		col := int((p.X - minX) / rangeX * float64(width-1))
		row := height - 1 - int((p.Y-minY)/rangeY*float64(height-1))
		if col >= 0 && col < width && row >= 0 && row < height {
			grid[row][col]++
		}
	}

	dotStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00D46A"))

	var b strings.Builder
	b.Grow(height * (width + 5))
	for _, row := range grid {
		b.WriteString("  │")
		for _, cell := range row {
			switch {
			case cell == 0:
				b.WriteByte(' ')
			case cell == 1:
				b.WriteString(dotStyle.Render("·"))
			case cell <= 3:
				b.WriteString(dotStyle.Render("●"))
			default:
				b.WriteString(dotStyle.Render("◉"))
			}
		}
		b.WriteByte('\n')
	}

	out := b.String()
	return strings.TrimSuffix(out, "\n")
}
