package store

import (
	"strings"
	"time"
)

const dateOnlyLayout = "2006-01-02"

// NormalizeBounds converts optional from/to filters into RFC3339 bounds suitable
// for lexicographic comparison against stored ISO-8601 timestamps.
//
// Date-only values (YYYY-MM-DD) are treated as inclusive calendar days in UTC:
// from becomes 00:00:00Z, to becomes 23:59:59Z. Values that already look like
// timestamps (contain "T") are left unchanged. Empty values stay empty.
func NormalizeBounds(from, to string) (string, string) {
	return normalizeBound(from, false), normalizeBound(to, true)
}

func normalizeBound(value string, endOfDay bool) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "T") {
		return value
	}
	parsed, err := time.Parse(dateOnlyLayout, value)
	if err != nil {
		return value
	}
	if endOfDay {
		return parsed.Add(24*time.Hour - time.Second).Format(time.RFC3339)
	}
	return parsed.Format(time.RFC3339)
}
