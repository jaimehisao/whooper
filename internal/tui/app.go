package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var tabNames = []string{"Dashboard", "Recovery", "Sleep", "Workouts", "Correlations"}

type syncDoneMsg struct{ err error }

type App struct {
	syncFunc  func() error
	keys      KeyMap
	activeTab int
	views     [5]tea.Model
	syncing   bool
	syncMsg   string
	width     int
	height    int
}

func NewApp(syncFn func() error) *App {
	return &App{
		syncFunc: syncFn,
		keys:     DefaultKeyMap(),
	}
}

// SetViews sets the 5 tab views. Called externally since views import tui package.
func (a *App) SetViews(dashboard, recovery, sleep, workouts, correlations tea.Model) {
	a.views = [5]tea.Model{dashboard, recovery, sleep, workouts, correlations}
}

func (a *App) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, v := range a.views {
		if v != nil {
			cmds = append(cmds, v.Init())
		}
	}
	return tea.Batch(cmds...)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case syncDoneMsg:
		a.syncing = false
		if msg.err != nil {
			a.syncMsg = fmt.Sprintf("Sync error: %v", msg.err)
		} else {
			a.syncMsg = "Sync complete!"
			if v, ok := a.views[a.activeTab].(interface{ Refresh() tea.Cmd }); ok {
				return a, v.Refresh()
			}
		}
		return a, nil

	case tea.KeyMsg:
		if key.Matches(msg, a.keys.Quit) {
			return a, tea.Quit
		}

		newTab := -1
		switch {
		case key.Matches(msg, a.keys.Tab1):
			newTab = 0
		case key.Matches(msg, a.keys.Tab2):
			newTab = 1
		case key.Matches(msg, a.keys.Tab3):
			newTab = 2
		case key.Matches(msg, a.keys.Tab4):
			newTab = 3
		case key.Matches(msg, a.keys.Tab5):
			newTab = 4
		case key.Matches(msg, a.keys.NextTab):
			newTab = (a.activeTab + 1) % 5
		case key.Matches(msg, a.keys.Sync):
			if !a.syncing && a.syncFunc != nil {
				a.syncing = true
				a.syncMsg = "Syncing..."
				return a, func() tea.Msg {
					err := a.syncFunc()
					return syncDoneMsg{err: err}
				}
			}
			return a, nil
		}

		if newTab >= 0 && newTab != a.activeTab {
			a.activeTab = newTab
			a.syncMsg = ""
			if v, ok := a.views[a.activeTab].(interface{ Refresh() tea.Cmd }); ok {
				return a, v.Refresh()
			}
			return a, nil
		}
	}

	// Pass to active view
	if a.views[a.activeTab] != nil {
		var cmd tea.Cmd
		a.views[a.activeTab], cmd = a.views[a.activeTab].Update(msg)
		return a, cmd
	}

	return a, nil
}

func (a *App) View() string {
	var tabs []string
	for i, name := range tabNames {
		label := fmt.Sprintf(" %d:%s ", i+1, name)
		if i == a.activeTab {
			tabs = append(tabs, ActiveTabStyle.Render(label))
		} else {
			tabs = append(tabs, TabStyle.Render(label))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	content := ""
	if a.views[a.activeTab] != nil {
		content = a.views[a.activeTab].View()
	}

	statusParts := []string{
		MutedStyle.Render("q quit"),
		MutedStyle.Render("s sync"),
		MutedStyle.Render("1-5 tabs"),
		MutedStyle.Render("< > range"),
	}
	if a.syncMsg != "" {
		if a.syncing {
			statusParts = append(statusParts, YellowStyle.Render(a.syncMsg))
		} else {
			statusParts = append(statusParts, GreenStyle.Render(a.syncMsg))
		}
	}
	statusBar := lipgloss.JoinHorizontal(lipgloss.Top, statusParts[0], "  ", statusParts[1], "  ", statusParts[2], "  ", statusParts[3])
	if len(statusParts) > 4 {
		statusBar += "  " + statusParts[4]
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		tabBar,
		"",
		content,
		"",
		statusBar,
	)
}

func RunApp(app *App) error {
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
