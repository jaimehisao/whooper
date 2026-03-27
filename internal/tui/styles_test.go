package tui

import "testing"

func TestRecoveryColor(t *testing.T) {
	if RecoveryColor(80).GetForeground() != GreenStyle.GetForeground() {
		t.Fatal("high score should use GreenStyle")
	}
	if RecoveryColor(50).GetForeground() != YellowStyle.GetForeground() {
		t.Fatal("mid score should use YellowStyle")
	}
	if RecoveryColor(20).GetForeground() != RedStyle.GetForeground() {
		t.Fatal("low score should use RedStyle")
	}
}
