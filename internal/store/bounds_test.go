package store

import "testing"

func TestNormalizeBounds_DateOnlyInclusive(t *testing.T) {
	from, to := NormalizeBounds("2024-01-08", "2024-01-08")
	if from != "2024-01-08T00:00:00Z" {
		t.Fatalf("from = %q, want 2024-01-08T00:00:00Z", from)
	}
	if to != "2024-01-08T23:59:59Z" {
		t.Fatalf("to = %q, want 2024-01-08T23:59:59Z", to)
	}
}

func TestNormalizeBounds_TimestampPassthrough(t *testing.T) {
	from, to := NormalizeBounds("2024-01-08T07:00:00Z", "2024-01-08T12:00:00Z")
	if from != "2024-01-08T07:00:00Z" || to != "2024-01-08T12:00:00Z" {
		t.Fatalf("passthrough failed: from=%q to=%q", from, to)
	}
}

func TestNormalizeBounds_Empty(t *testing.T) {
	from, to := NormalizeBounds("", "")
	if from != "" || to != "" {
		t.Fatalf("empty bounds should stay empty: from=%q to=%q", from, to)
	}
}

func TestNormalizeBounds_IncludesLastDayTimestamps(t *testing.T) {
	_, to := NormalizeBounds("", "2024-01-08")
	ts := "2024-01-08T07:00:00Z"
	if ts > to {
		t.Fatalf("timestamp %q should be <= inclusive end bound %q", ts, to)
	}
}
