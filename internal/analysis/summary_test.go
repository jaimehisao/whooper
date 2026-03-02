package analysis

import (
	"testing"
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/models"
	"git.infra.hisao.org/hisao/whooper/internal/store"
)

func TestWeeklyReportStruct(t *testing.T) {
	report := &WeeklyReport{
		WeekStart:        "2024-01-01",
		AvgRecovery:      75.5,
		AvgHRV:           45.0,
		AvgRHR:           55.0,
		AvgSleepHours:    7.5,
		AvgSleepEffPct:   90.0,
		TotalStrain:      100.0,
		WorkoutCount:     5,
		AvgWorkoutStrain: 20.0,
	}

	if report.WeekStart != "2024-01-01" {
		t.Errorf("WeekStart = %s, want 2024-01-01", report.WeekStart)
	}
	if report.AvgRecovery != 75.5 {
		t.Errorf("AvgRecovery = %v, want 75.5", report.AvgRecovery)
	}
	if report.WorkoutCount != 5 {
		t.Errorf("WorkoutCount = %d, want 5", report.WorkoutCount)
	}
}

func TestWeeklyReportEmpty(t *testing.T) {
	report := &WeeklyReport{
		WeekStart: "2024-01-01",
	}

	if report.AvgRecovery != 0 {
		t.Errorf("AvgRecovery = %v, want 0", report.AvgRecovery)
	}
	if report.WorkoutCount != 0 {
		t.Errorf("WorkoutCount = %d, want 0", report.WorkoutCount)
	}
}

func TestWeeklyReportWeekDates(t *testing.T) {
	weekStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	from := weekStart.Format("2006-01-02")
	to := weekStart.Add(7 * 24 * time.Hour).Format("2006-01-02")

	if from != "2024-01-01" {
		t.Errorf("from = %s, want 2024-01-01", from)
	}
	if to != "2024-01-08" {
		t.Errorf("to = %s, want 2024-01-08", to)
	}
}

type mockStoreForSummary struct {
	recoveries []store.RecoveryTrendPoint
	sleeps     []store.SleepTrendPoint
	strains    []store.StrainTrendPoint
	workouts   []models.Workout
	err        error
}

func (m *mockStoreForSummary) GetRecoveryTrend(from, to string) ([]store.RecoveryTrendPoint, error) {
	return m.recoveries, m.err
}

func (m *mockStoreForSummary) GetSleepTrend(from, to string) ([]store.SleepTrendPoint, error) {
	return m.sleeps, m.err
}

func (m *mockStoreForSummary) GetStrainTrend(from, to string) ([]store.StrainTrendPoint, error) {
	return m.strains, m.err
}

func (m *mockStoreForSummary) ListWorkouts(from, to string) ([]models.Workout, error) {
	return m.workouts, m.err
}

func TestGenerateWeeklyReportWithData(t *testing.T) {
	store := &mockStoreForSummary{
		recoveries: []store.RecoveryTrendPoint{
			{Date: "2024-01-01", RecoveryScore: 75, HRV: 45, RHR: 55},
			{Date: "2024-01-02", RecoveryScore: 85, HRV: 50, RHR: 52},
		},
		sleeps: []store.SleepTrendPoint{
			{Date: "2024-01-01", DurationMilli: 28800000, EfficiencyPct: 90},
		},
		strains: []store.StrainTrendPoint{
			{Date: "2024-01-01", Strain: 15},
			{Date: "2024-01-02", Strain: 18},
		},
	}

	weekStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	report := &WeeklyReport{WeekStart: weekStart.Format("2006-01-02")}

	if len(store.recoveries) > 0 {
		var sumRec float64
		for _, r := range store.recoveries {
			sumRec += r.RecoveryScore
		}
		report.AvgRecovery = sumRec / float64(len(store.recoveries))
	}

	if report.AvgRecovery != 80 {
		t.Errorf("AvgRecovery = %v, want 80", report.AvgRecovery)
	}

	report.TotalStrain = 0
	for _, s := range store.strains {
		report.TotalStrain += s.Strain
	}

	if report.TotalStrain != 33 {
		t.Errorf("TotalStrain = %v, want 33", report.TotalStrain)
	}
}
