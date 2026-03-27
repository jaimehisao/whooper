package tui

import "testing"

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	if len(km.Tab1.Keys()) == 0 || km.Tab1.Keys()[0] != "1" {
		t.Fatalf("Tab1 keys = %v, want first key 1", km.Tab1.Keys())
	}
	if len(km.Quit.Keys()) < 2 {
		t.Fatalf("Quit should include q and ctrl+c")
	}
	if len(km.Sync.Keys()) == 0 || km.Sync.Keys()[0] != "s" {
		t.Fatalf("Sync keys = %v, want s", km.Sync.Keys())
	}
	if len(km.NextTab.Keys()) == 0 || km.NextTab.Keys()[0] != "tab" {
		t.Fatalf("NextTab keys = %v, want tab", km.NextTab.Keys())
	}
	if len(km.PrevTab.Keys()) == 0 || km.PrevTab.Keys()[0] != "shift+tab" {
		t.Fatalf("PrevTab keys = %v, want shift+tab", km.PrevTab.Keys())
	}
}
