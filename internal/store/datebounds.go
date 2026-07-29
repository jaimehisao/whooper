package store

import (
	"fmt"
	"time"
)

const dateOnlyLayout = "2006-01-02"

// ParseDateOnly parses a YYYY-MM-DD date. Empty string returns zero time.
func ParseDateOnly(name, value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(dateOnlyLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid '%s' date %q: use YYYY-MM-DD", name, value)
	}
	return parsed, nil
}

// ValidateDateRange ensures from/to are valid and to is not before from.
func ValidateDateRange(from, to string) error {
	fromDate, err := ParseDateOnly("from", from)
	if err != nil {
		return err
	}
	toDate, err := ParseDateOnly("to", to)
	if err != nil {
		return err
	}
	if !fromDate.IsZero() && !toDate.IsZero() && toDate.Before(fromDate) {
		return fmt.Errorf("to date %q cannot be before from date %q", to, from)
	}
	return nil
}

// DateBounds converts inclusive YYYY-MM-DD from/to into RFC3339 timestamp bounds
// suitable for comparing against RFC3339 columns. The to bound includes the entire day.
func DateBounds(from, to string) (fromBound, toBound string, err error) {
	if err := ValidateDateRange(from, to); err != nil {
		return "", "", err
	}
	if from != "" {
		fromDate, _ := ParseDateOnly("from", from)
		fromBound = fromDate.Format(time.RFC3339)
	}
	if to != "" {
		toDate, _ := ParseDateOnly("to", to)
		toBound = toDate.Add(24*time.Hour - time.Second).Format(time.RFC3339)
	}
	return fromBound, toBound, nil
}

// ExpandToBound expands a bare YYYY-MM-DD or already-timestamp to to an inclusive end-of-day bound.
// If value already contains "T", it is returned unchanged.
func ExpandToBound(to string) (string, error) {
	if to == "" {
		return "", nil
	}
	if len(to) > 10 && to[10] == 'T' {
		return to, nil
	}
	_, toBound, err := DateBounds("", to)
	return toBound, err
}

// ExpandFromBound expands a bare YYYY-MM-DD to start-of-day RFC3339.
func ExpandFromBound(from string) (string, error) {
	if from == "" {
		return "", nil
	}
	if len(from) > 10 && from[10] == 'T' {
		return from, nil
	}
	fromBound, _, err := DateBounds(from, "")
	return fromBound, err
}

// StartOfDayBound is an alias for ExpandFromBound.
func StartOfDayBound(from string) (string, error) { return ExpandFromBound(from) }

// EndOfDayBound is an alias for ExpandToBound.
func EndOfDayBound(to string) (string, error) { return ExpandToBound(to) }

// ExpandRangeBounds expands optional from/to filters for RFC3339 column comparisons.
func ExpandRangeBounds(from, to string) (fromBound, toBound string, err error) {
	fromBound, err = ExpandFromBound(from)
	if err != nil {
		return "", "", err
	}
	toBound, err = ExpandToBound(to)
	if err != nil {
		return "", "", err
	}
	return fromBound, toBound, nil
}

// localDateSQL returns a SQLite expression that buckets a timestamp column by local
// calendar day when timezone_offset is present, otherwise UTC date(col).
func localDateSQL(col string) string {
	return fmt.Sprintf(
		`CASE WHEN timezone_offset IS NOT NULL AND timezone_offset != '' THEN date(datetime(%s, timezone_offset)) ELSE date(%s) END`,
		col, col,
	)
}

func appendListBounds(query string, args []any, col, from, to string) (string, []any, error) {
	fromBound, err := ExpandFromBound(from)
	if err != nil {
		return "", nil, err
	}
	toBound, err := ExpandToBound(to)
	if err != nil {
		return "", nil, err
	}
	if fromBound != "" {
		query += ` AND ` + col + ` >= ?`
		args = append(args, fromBound)
	}
	if toBound != "" {
		query += ` AND ` + col + ` <= ?`
		args = append(args, toBound)
	}
	return query, args, nil
}
