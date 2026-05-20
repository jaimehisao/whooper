package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestApp_SyncStateTransitions(t *testing.T) {
	app := NewApp(func() error { return nil })
	mock := &mockModel{}
	app.SetViews(mock, mock, mock, mock, mock)

	// Start sync
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if !app.syncing || app.syncMsg != "Syncing..." {
		t.Errorf("Syncing not started correctly: %v, %q", app.syncing, app.syncMsg)
	}
	if !strings.Contains(app.View(), "* Syncing...") {
		t.Error("View should show syncing indicator")
	}

	// Sync success
	app.Update(syncDoneMsg{err: nil})
	if app.syncing || app.syncMsg != "Sync complete!" {
		t.Errorf("Sync success not handled correctly: %v, %q", app.syncing, app.syncMsg)
	}
	if !strings.Contains(app.View(), "V Sync complete!") {
		t.Error("View should show success indicator")
	}

	// Sync error
	app.Update(syncDoneMsg{err: errors.New("network error")})
	if app.syncing || !app.syncErr || !strings.Contains(app.syncMsg, "network error") {
		t.Errorf("Sync error not handled correctly: %v, %v, %q", app.syncing, app.syncErr, app.syncMsg)
	}
	if !strings.Contains(app.View(), "X Sync error: network error") {
		t.Error("View should show error indicator")
	}

	// Clear tick
	app.Update(clearSyncMsg{})
	if app.syncMsg != "" {
		t.Error("Sync message should be cleared")
	}
}

func TestApp_WindowResizePropagation(t *testing.T) {
	app := NewApp(nil)
	mock := &mockModel{}
	app.SetViews(mock, mock, mock, mock, mock)

	msg := tea.WindowSizeMsg{Width: 120, Height: 50}
	app.Update(msg)

	if app.width != 120 || app.height != 50 {
		t.Errorf("App size not updated: %dx%d", app.width, app.height)
	}
	// Note: in a real test we'd check if views were updated, but our mock doesn't store size.
	// But app_test.go already checks this basic propagation.
}

func TestApp_TabNavigation(t *testing.T) {
	app := NewApp(nil)
	mock := &mockModel{}
	app.SetViews(mock, mock, mock, mock, mock)

	// Number keys
	for i := 0; i < 5; i++ {
		app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune('1' + i)}})
		if app.activeTab != i {
			t.Errorf("Failed to switch to tab %d via key %d", i, i+1)
		}
	}

	// Tab / Shift+Tab
	app.activeTab = 0
	app.Update(tea.KeyMsg{Type: tea.KeyTab})
	if app.activeTab != 1 {
		t.Errorf("Tab key: expected tab 1, got %d", app.activeTab)
	}
	app.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if app.activeTab != 0 {
		t.Errorf("Shift+Tab key: expected tab 0, got %d", app.activeTab)
	}
}
