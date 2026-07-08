package tui

import (
	"errors"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type mockModel struct {
	updated bool
	viewed  bool
}

func (m *mockModel) Init() tea.Cmd { return nil }
func (m *mockModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.updated = true
	return m, nil
}
func (m *mockModel) View() string {
	m.viewed = true
	return "mock view"
}
func (m *mockModel) Refresh() tea.Cmd { return nil }

func TestApp(t *testing.T) {
	app := NewApp(func() error { return nil })
	mock := &mockModel{}
	app.SetViews(mock, mock, mock, mock, mock)

	// Test Init
	app.Init()

	// Test Update WindowSize
	msg := tea.WindowSizeMsg{Width: 100, Height: 40}
	app.Update(msg)
	if app.width != 100 || app.height != 40 {
		t.Errorf("Expected size 100x40, got %dx%d", app.width, app.height)
	}

	// Test Tab switching
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	if app.activeTab != 1 {
		t.Errorf("Expected tab 1, got %d", app.activeTab)
	}

	// Test Sync trigger
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if !app.syncing {
		t.Error("Expected syncing to be true")
	}

	// Test Sync done
	app.Update(syncDoneMsg{err: nil})
	if app.syncing {
		t.Error("Expected syncing to be false")
	}
	if app.syncMsg != "Sync complete!" {
		t.Errorf("Unexpected sync msg: %s", app.syncMsg)
	}

	// Test Sync error
	app.Update(syncDoneMsg{err: errors.New("fail")})
	if !app.syncErr {
		t.Error("Expected syncErr to be true")
	}

	// Test View
	view := app.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}

func TestApp_UpdateMisc(t *testing.T) {
	app := NewApp(func() error { return nil })
	mock := &mockModel{}
	app.SetViews(mock, mock, mock, mock, mock)

	// Test navigation keys passed to views
	app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !mock.updated {
		t.Error("Expected view Update to be called on Esc")
	}

	// Test clearSyncMsg
	app.syncing = false
	app.syncMsg = "Done"
	app.Update(clearSyncMsg{})
	if app.syncMsg != "" {
		t.Error("Expected syncMsg to be cleared")
	}

	// Test Tab names and Styles smoke test
	app.View()
}

func TestApp_UpdateAdditionalBranches(t *testing.T) {
	app := NewApp(nil)
	mock := &mockModel{}
	app.SetViews(mock, mock, mock, mock, mock)

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = model.(*App)
	if app.activeTab != 1 {
		t.Fatalf("activeTab after tab = %d, want 1", app.activeTab)
	}

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	app = model.(*App)
	if app.activeTab != 0 {
		t.Fatalf("activeTab after shift+tab = %d, want 0", app.activeTab)
	}

	app.syncing = true
	app.syncMsg = "Syncing..."
	app.Update(clearSyncMsg{})
	if app.syncMsg != "Syncing..." {
		t.Fatalf("clearSyncMsg cleared active sync message")
	}

	app.syncing = false
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if app.syncing {
		t.Fatalf("sync without syncFunc should not start")
	}
	if app.syncMsg == "" || !app.syncErr {
		t.Fatalf("sync without syncFunc should show login-required error, got msg=%q err=%v", app.syncMsg, app.syncErr)
	}
	if cmd == nil {
		t.Fatal("sync without syncFunc should schedule clear tick")
	}

	_, cmd = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("quit key should return a command")
	}
}

func TestKeyMap(t *testing.T) {
	km := DefaultKeyMap()
	if !key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}, km.Quit) {
		t.Error("Expected q to match Quit")
	}
}
