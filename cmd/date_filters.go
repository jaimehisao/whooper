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
		// Include the entire day by setting to the last nanosecond of the day
		toBound = toDate.Add(24*time.Hour - time.Nanosecond).Format(time.RFC3339Nano)
	}

	return fromBound, toBound, nil
}
