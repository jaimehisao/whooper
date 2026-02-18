package components

import (
	"strings"
	"testing"
)

func TestTable_Empty(t *testing.T) {
	result := Table(nil, nil, nil)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestTable_HeadersOnly(t *testing.T) {
	result := Table([]string{"A", "B"}, nil, nil)
	if result == "" {
		t.Error("should render headers")
	}
	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	if len(lines) != 2 { // header + separator
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestTable_WithRows(t *testing.T) {
	headers := []string{"Name", "Value"}
	rows := [][]string{{"foo", "1"}, {"bar", "2"}}
	result := Table(headers, rows, []int{10, 8})
	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	if len(lines) != 4 { // header + separator + 2 rows
		t.Errorf("expected 4 lines, got %d", len(lines))
	}
}

func TestTable_ShortRow(t *testing.T) {
	headers := []string{"A", "B", "C"}
	rows := [][]string{{"x"}} // row shorter than headers
	result := Table(headers, rows, nil)
	if result == "" {
		t.Error("should handle short rows gracefully")
	}
}

func TestHighlightedTable_CursorRow(t *testing.T) {
	headers := []string{"Name"}
	rows := [][]string{{"a"}, {"b"}, {"c"}}
	result := HighlightedTable(headers, rows, []int{10}, 1)
	if result == "" {
		t.Error("should render highlighted table")
	}
}

func TestHighlightedTable_Empty(t *testing.T) {
	result := HighlightedTable(nil, nil, nil, 0)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"hello world", 8, "hello..."},
		{"ab", 2, "ab"},
		{"abcd", 3, "abc"},
		{"abcdef", 4, "a..."},
	}
	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestColWidth_Default(t *testing.T) {
	if w := colWidth(nil, 0); w != 12 {
		t.Errorf("expected default 12, got %d", w)
	}
	if w := colWidth([]int{20}, 0); w != 20 {
		t.Errorf("expected 20, got %d", w)
	}
	if w := colWidth([]int{20}, 5); w != 12 {
		t.Errorf("expected default 12 for out-of-range, got %d", w)
	}
}
