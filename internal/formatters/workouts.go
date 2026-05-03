package formatters

import (
	"fmt"
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/models"
)

type WorkoutDisplayData struct {
	Workouts    []WorkoutRow
	TotalStrain float64
	TotalCount  int
	AvgStrain   float64
	HasData     bool
}

type WorkoutRow struct {
	ID           string
	Date         string
	Strain       float64
	DurationMins int
	SportName    string
	HasScore     bool
}

var sportIDToName = map[int]string{
	1:  "Running",
	2:  "Cycling",
	3:  "Swimming",
	4:  "Weightlifting",
	5:  "HIIT",
	6:  "Yoga",
	7:  "Rowing",
	8:  "Stair Climb",
	9:  "Hiking",
	10: "Skiing",
	11: "Snowboarding",
	12: "Tennis",
	13: "Basketball",
	14: "Soccer",
	15: "Baseball",
	16: "Football",
	17: "Golf",
	18: "Volleyball",
	19: "Rock Climbing",
	20: "Other",
}

func FormatWorkoutData(workouts []models.Workout) WorkoutDisplayData {
	result := WorkoutDisplayData{
		HasData: len(workouts) > 0,
	}

	if len(workouts) == 0 {
		return result
	}

	var totalStrain float64
	for _, w := range workouts {
		row := WorkoutRow{
			ID:       w.ID,
			Date:     formatDate(w.Start),
			HasScore: w.Score != nil,
		}

		if w.Score != nil {
			row.Strain = w.Score.Strain
			totalStrain += w.Score.Strain
		}

		row.DurationMins = calcDurationMins(w.Start, w.End)
		row.SportName = sportIDToName[w.SportID]
		if row.SportName == "" {
			row.SportName = "Unknown"
		}

		result.Workouts = append(result.Workouts, row)
	}

	result.TotalCount = len(workouts)
	result.TotalStrain = totalStrain
	if len(workouts) > 0 {
		result.AvgStrain = totalStrain / float64(len(workouts))
	}

	return result
}

func FormatWorkoutTable(workouts []models.Workout, width int) string {
	if len(workouts) == 0 {
		return "No workouts found"
	}

	header := fmt.Sprintf("%-6s %-12s %-8s %-6s %s",
		"ID", "Date", "Duration", "Strain", "Sport")

	rows := []string{header}
	for _, w := range workouts {
		duration := calcDurationMins(w.Start, w.End)
		strain := "N/A"
		if w.Score != nil {
			strain = fmt.Sprintf("%.1f", w.Score.Strain)
		}
		sport := sportIDToName[w.SportID]
		if sport == "" {
			sport = "Unknown"
		}

		row := fmt.Sprintf("%-6s %-12s %-8d %-6s %s",
			w.ID, formatDate(w.Start), duration, strain, sport)
		rows = append(rows, row)
	}

	return joinStrings(rows, "\n")
}

func FormatWorkoutSummary(workouts []models.Workout) string {
	if len(workouts) == 0 {
		return "No workouts this period"
	}

	var totalStrain, count float64
	for _, w := range workouts {
		if w.Score != nil {
			totalStrain += w.Score.Strain
			count++
		}
	}

	avgStrain := 0.0
	if count > 0 {
		avgStrain = totalStrain / count
	}

	return fmt.Sprintf("%d workouts | Total strain: %.1f | Avg: %.1f",
		len(workouts), totalStrain, avgStrain)
}

func calcDurationMins(start, end string) int {
	startTime, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return 0
	}
	endTime, err := time.Parse(time.RFC3339, end)
	if err != nil {
		return 0
	}
	duration := endTime.Sub(startTime)
	return int(duration.Minutes())
}

func formatDate(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		if len(s) >= 10 {
			return s[:10]
		}
		return s
	}
	return t.Format("2006-01-02")
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

func WorkoutDateRange(days int) (string, string) {
	now := time.Now().UTC()
	to := now.Format("2006-01-02")
	from := now.Add(-time.Duration(days) * 24 * time.Hour).Format("2006-01-02")
	return from, to
}

func SportName(sportID int) string {
	return sportIDToName[sportID]
}
