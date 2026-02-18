package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the application.
type KeyMap struct {
	Tab1       key.Binding
	Tab2       key.Binding
	Tab3       key.Binding
	Tab4       key.Binding
	Tab5       key.Binding
	Quit       key.Binding
	Sync       key.Binding
	Left       key.Binding
	Right      key.Binding
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding
	NextTab    key.Binding
	BracketL   key.Binding
	BracketR   key.Binding
}

// DefaultKeyMap returns the default set of keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Tab1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "dashboard"),
		),
		Tab2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "recovery"),
		),
		Tab3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "sleep"),
		),
		Tab4: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "workouts"),
		),
		Tab5: key.NewBinding(
			key.WithKeys("5"),
			key.WithHelp("5", "correlations"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Sync: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sync"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "<"),
			key.WithHelp("<", "prev range"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", ">"),
			key.WithHelp(">", "next range"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("up", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("down", "move down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next tab"),
		),
		BracketL: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "prev Y metric"),
		),
		BracketR: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next Y metric"),
		),
	}
}
