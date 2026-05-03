package formatters

import (
	"strings"
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/models"
)

func TestFormatWorkoutData_Empty(t *testing.T) {
	result := FormatWorkoutData(nil)
	if result.HasData {
		t.Error("expected HasData=false for nil")
	}

	result = FormatWorkoutData([]models.Workout{})
	if result.HasData {
		t.Error("expected HasData=false for empty")
	}
}

func TestFormatWorkoutData_WithData(t *testing.T) {
	workouts := []models.Workout{
		{
			ID:      1,
			Start:   "2024-01-15T10:00:00Z",
			End:     "2024-01-15T11:30:00Z",
			SportID: 1,
			Score: &models.WorkoutScore{
				Strain: 15.5,
			},
		},
		{
			ID:      2,
			Start:   "2024-01-16T14:00:00Z",
			End:     "2024-01-16T15:00:00Z",
			SportID: 4,
			Score:   nil,
		},
	}

	result := FormatWorkoutData(workouts)

	if !result.HasData {
		t.Error("expected HasData=true")
	}
	if result.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", result.TotalCount)
	}
	if result.TotalStrain != 15.5 {
		t.Errorf("TotalStrain = %v, want 15.5", result.TotalStrain)
	}
	// AvgStrain is calculated using total workouts (2), not just scored (1)
	// So: 15.5 / 2 = 7.75
	if result.AvgStrain != 7.75 {
		t.Errorf("AvgStrain = %v, want 7.75", result.AvgStrain)
	}
}

func TestFormatWorkoutTable(t *testing.T) {
	workouts := []models.Workout{
		{
			ID:      7,
			Start:   "2024-01-15T10:00:00Z",
			End:     "2024-01-15T11:30:00Z",
			SportID: 1,
			Score:   &models.WorkoutScore{Strain: 15.5},
		},
		{
			ID:      8,
			Start:   "2024-01-16T10:00:00Z",
			End:     "bad-end",
			SportID: 999,
		},
	}

	table := FormatWorkoutTable(workouts, 80)
	for _, want := range []string{"ID", "Date", "Duration", "Strain", "Sport", "2024-01-15", "15.5", "Running", "N/A", "Unknown"} {
		if !strings.Contains(table, want) {
			t.Fatalf("FormatWorkoutTable() missing %q in:\n%s", want, table)
		}
	}
}

func TestFormatWorkoutTable_Empty(t *testing.T) {
	if got := FormatWorkoutTable(nil, 80); got != "No workouts found" {
		t.Fatalf("FormatWorkoutTable(nil) = %q, want no workouts message", got)
	}
}

func TestFormatWorkoutSummary(t *testing.T) {
	workouts := []models.Workout{
		{ID: 1, Score: &models.WorkoutScore{Strain: 10}},
		{ID: 2, Score: &models.WorkoutScore{Strain: 20}},
		{ID: 3},
	}

	got := FormatWorkoutSummary(workouts)
	want := "3 workouts | Total strain: 30.0 | Avg: 15.0"
	if got != want {
		t.Fatalf("FormatWorkoutSummary() = %q, want %q", got, want)
	}
}

func TestFormatWorkoutSummary_EmptyAndUnscored(t *testing.T) {
	if got := FormatWorkoutSummary(nil); got != "No workouts this period" {
		t.Fatalf("FormatWorkoutSummary(nil) = %q", got)
	}

	got := FormatWorkoutSummary([]models.Workout{{ID: 1}, {ID: 2}})
	want := "2 workouts | Total strain: 0.0 | Avg: 0.0"
	if got != want {
		t.Fatalf("FormatWorkoutSummary(unscored) = %q, want %q", got, want)
	}
}

func TestFormatWorkoutData_AllScored(t *testing.T) {
	workouts := []models.Workout{
		{
			ID:      1,
			SportID: 1,
			Score:   &models.WorkoutScore{Strain: 10},
		},
		{
			ID:      2,
			SportID: 2,
			Score:   &models.WorkoutScore{Strain: 20},
		},
	}

	result := FormatWorkoutData(workouts)

	if result.AvgStrain != 15 {
		t.Errorf("AvgStrain = %v, want 15", result.AvgStrain)
	}
}

func TestCalcDurationMins(t *testing.T) {
	result := calcDurationMins("2024-01-15T10:00:00Z", "2024-01-15T11:30:00Z")
	if result != 90 {
		t.Errorf("calcDurationMins() = %d, want 90", result)
	}

	result = calcDurationMins("invalid", "2024-01-15T11:00:00Z")
	if result != 0 {
		t.Errorf("calcDurationMins(invalid) = %d, want 0", result)
	}
}

func TestFormatDate(t *testing.T) {
	result := formatDate("2024-01-15T10:30:00Z")
	if result != "2024-01-15" {
		t.Errorf("formatDate() = %q, want 2024-01-15", result)
	}

	result = formatDate("invalid")
	if result == "" {
		t.Error("expected non-empty for invalid input")
	}
}

func TestJoinStrings(t *testing.T) {
	result := joinStrings([]string{"a", "b", "c"}, ",")
	if result != "a,b,c" {
		t.Errorf("joinStrings() = %q, want a,b,c", result)
	}

	result = joinStrings([]string{"single"}, "-")
	if result != "single" {
		t.Errorf("joinStrings(single) = %q, want single", result)
	}

	result = joinStrings(nil, ",")
	if result != "" {
		t.Errorf("joinStrings(nil) = %q, want empty", result)
	}
}

func TestSportName(t *testing.T) {
	if SportName(1) != "Running" {
		t.Errorf("SportName(1) = %s, want Running", SportName(1))
	}
	if SportName(2) != "Cycling" {
		t.Errorf("SportName(2) = %s, want Cycling", SportName(2))
	}
	if SportName(999) != "" {
		t.Errorf("SportName(999) = %s, want empty", SportName(999))
	}
}

func TestWorkoutDateRange(t *testing.T) {
	from, to := WorkoutDateRange(7)
	if from == "" || to == "" {
		t.Error("expected non-empty dates")
	}
	if from >= to {
		t.Errorf("from (%s) should be before to (%s)", from, to)
	}
}
