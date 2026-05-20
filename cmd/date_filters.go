package cmd

import (
	"fmt"
	"time"
)

const dateOnlyLayout = "2006-01-02"

func parseDateOnly(name, value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(dateOnlyLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid '%s' date %q: use YYYY-MM-DD", name, value)
	}
	return parsed, nil
}

func validateDateRange(from, to string) error {
	fromDate, err := parseDateOnly("from", from)
	if err != nil {
		return err
	}
	toDate, err := parseDateOnly("to", to)
	if err != nil {
		return err
	}
	if !fromDate.IsZero() && !toDate.IsZero() && toDate.Before(fromDate) {
		return fmt.Errorf("to date %q cannot be before from date %q", to, from)
	}
	return nil
}

func exportDateBounds(from, to string) (string, string, error) {
	if err := validateDateRange(from, to); err != nil {
		return "", "", err
	}

	fromBound := ""
	if from != "" {
		fromDate, _ := parseDateOnly("from", from)
		fromBound = fromDate.Format(time.RFC3339)
	}

	toBound := ""
	if to != "" {
		toDate, _ := parseDateOnly("to", to)
		// Include the entire day. Using 23:59:59Z is robust for string comparisons
		// against RFC3339 timestamps (with or without fractional seconds) because
		// '.' and '+' and '-' all sort before 'Z'.
		toBound = toDate.Add(24*time.Hour - time.Second).Format(time.RFC3339)
	}

	return fromBound, toBound, nil
}
