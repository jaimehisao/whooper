package formatters

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestFormatFloatSimple_Negative(t *testing.T) {
	got := formatFloatSimple(-0.65)
	if got != "-0.65" {
		t.Fatalf("formatFloatSimple(-0.65) = %q, want -0.65", got)
	}
	got = formatFloatSimple(-1.2)
	if got != "-1.20" {
		t.Fatalf("formatFloatSimple(-1.2) = %q, want -1.20", got)
	}

	desc := CorrelationDescription("recovery", "hrv", -0.65)
	if !strings.Contains(desc, "r=-0.65") {
		t.Fatalf("description missing negative r: %q", desc)
	}
	if strings.Contains(desc, "r=-.") || strings.HasSuffix(desc, "(r=-)") {
		t.Fatalf("broken negative formatting: %q", desc)
	}
}

func TestSparkline_UsesRunes(t *testing.T) {
	data := []float64{0, 50, 100}
	s := sparkline(data, 3)
	if s == "" {
		t.Fatal("expected non-empty sparkline")
	}
	if utf8.RuneCountInString(s) != 3 {
		t.Fatalf("rune count = %d, want 3 (got %q)", utf8.RuneCountInString(s), s)
	}
	// Byte length of multi-byte block chars must exceed rune count if indexing by byte was wrong
	// (the old bug indexed into UTF-8 bytes). Ensure we only emit valid runes from the block set.
	blocks := []rune(" ▁▂▃▄▅▆▇█")
	for _, r := range s {
		found := false
		for _, b := range blocks {
			if r == b {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("unexpected rune %q in sparkline %q", string(r), s)
		}
	}
}
