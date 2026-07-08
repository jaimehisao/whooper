package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var tabNames = []string{"Dashboard", "Recovery", "Sleep", "Workouts", "Correlations"}

type syncDoneMsg struct{ err error }
type clearSyncMsg struct{}

type App struct {
	syncFunc  func() error
	keys      KeyMap
	activeTab int
	views     [5]tea.Model
	syncing   bool
	syncMsg   string
	syncErr   bool
	width     int
	height    int
}

func NewApp(syncFn func() error) *App {
	return &App{
		syncFunc: syncFn,
		keys:     DefaultKeyMap(),
	}
}

// Syncing reports whether an in-progress sync was started from the TUI.
func (a *App) Syncing() bool {
	return a.syncing
}

// SyncMessage returns the current sync status message (for tests/UX checks).
func (a *App) SyncMessage() string {
	return a.syncMsg
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
		// Propagate to all views
		var cmds []tea.Cmd
		for i, v := range a.views {
			if v != nil {
				var cmd tea.Cmd
				a.views[i], cmd = v.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
		return a, tea.Batch(cmds...)

	case syncDoneMsg:
		a.syncing = false
		if msg.err != nil {
			a.syncMsg = fmt.Sprintf("Sync error: %v", msg.err)
			a.syncErr = true
		} else {
			a.syncMsg = "Sync complete!"
			a.syncErr = false
		}
		// Refresh all views after sync
		var cmds []tea.Cmd
		for _, v := range a.views {
			if r, ok := v.(interface{ Refresh() tea.Cmd }); ok {
				cmds = append(cmds, r.Refresh())
			}
		}
		// Clear sync message after 5 seconds
		cmds = append(cmds, tea.Tick(5*time.Second, func(_ time.Time) tea.Msg {
			return clearSyncMsg{}
		}))
		return a, tea.Batch(cmds...)

	case clearSyncMsg:
		if !a.syncing {
			a.syncMsg = ""
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
		case key.Matches(msg, a.keys.PrevTab):
			newTab = (a.activeTab + 4) % 5
		case key.Matches(msg, a.keys.Sync):
			if a.syncFunc == nil {
				a.syncMsg = "Login required: run 'whooper login'"
				a.syncErr = true
				return a, tea.Tick(5*time.Second, func(_ time.Time) tea.Msg {
					return clearSyncMsg{}
				})
			}
			if !a.syncing {
				a.syncing = true
				a.syncMsg = "Syncing..."
				a.syncErr = false
				return a, func() tea.Msg {
					err := a.syncFunc()
					return syncDoneMsg{err: err}
				}
			}
			return a, nil
		}

		if newTab >= 0 && newTab != a.activeTab {
			a.activeTab = newTab
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

	// Status Bar — context-sensitive help (avoid conflicting key hints).
	helpParts := []string{
		MutedStyle.Render("q quit"),
	}
	if a.syncFunc != nil {
		helpParts = append(helpParts, MutedStyle.Render("s sync"))
	} else {
		helpParts = append(helpParts, MutedStyle.Render("s needs login"))
	}
	helpParts = append(helpParts, MutedStyle.Render("1-5 tabs"))
	switch a.activeTab {
	case 1, 2, 3: // Recovery, Sleep, Workouts
		helpParts = append(helpParts, MutedStyle.Render("< > range"))
	case 4: // Correlations
		helpParts = append(helpParts, MutedStyle.Render("< > X"), MutedStyle.Render("[ ] Y"))
	}
	if a.activeTab == 3 {
		helpParts = append(helpParts, MutedStyle.Render("j/k nav"), MutedStyle.Render("enter detail"))
	}

	helpBar := lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(helpParts, "  "))

	syncBar := ""
	if a.syncMsg != "" {
		if a.syncing {
			syncBar = YellowStyle.Render("* " + a.syncMsg)
		} else if a.syncErr {
			syncBar = RedStyle.Render("X " + a.syncMsg)
		} else {
			syncBar = GreenStyle.Render("V " + a.syncMsg)
		}
	}

	helpWidth := lipgloss.Width(helpBar)
	syncWidth := lipgloss.Width(syncBar)
	spaceCount := a.width - helpWidth - syncWidth
	if spaceCount < 0 {
		spaceCount = 0
	}

	statusBar := lipgloss.JoinHorizontal(lipgloss.Top,
		helpBar,
		strings.Repeat(" ", spaceCount),
		syncBar,
	)

	chrome := 4 // tab bar + blank lines + status bar
	contentHeight := a.height - chrome
	if contentHeight > 0 {
		content = clipToHeight(content, contentHeight)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		tabBar,
		"",
		content,
		"",
		statusBar,
	)
}

func clipToHeight(content string, maxLines int) string {
	if maxLines <= 0 {
		return content
	}
	lines := strings.Split(content, "\n")
	if len(lines) <= maxLines {
		return content
	}
	clipped := lines[:maxLines]
	if maxLines >= 1 {
		clipped[maxLines-1] = MutedStyle.Render("… (resize or switch tabs)")
	}
	return strings.Join(clipped, "\n")
}

func RunApp(app *App) error {
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
