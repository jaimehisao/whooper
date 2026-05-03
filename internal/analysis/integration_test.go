package analysis

import (
	"path/filepath"
	"testing"
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/models"
	"git.infra.hisao.org/hisao/whooper/internal/store"
)

func openAnalysisTestDB(t *testing.T) *store.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "analysis.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestGenerateWeeklyReport(t *testing.T) {
	db := openAnalysisTestDB(t)

	if err := db.SaveRecoveries([]models.Recovery{
		{
			CycleID: 1, SleepID: "11", UserID: 1,
			CreatedAt:  "2024-01-15T07:00:00Z",
			UpdatedAt:  "2024-01-15T07:00:00Z",
			ScoreState: "SCORED",
			Score: &models.RecoveryScore{
				RecoveryScore:    70,
				HRVRmssd:         42,
				RestingHeartRate: 56,
			},
		},
		{
			CycleID: 2, SleepID: "12", UserID: 1,
			CreatedAt:  "2024-01-16T07:00:00Z",
			UpdatedAt:  "2024-01-16T07:00:00Z",
			ScoreState: "SCORED",
			Score: &models.RecoveryScore{
				RecoveryScore:    90,
				HRVRmssd:         50,
				RestingHeartRate: 52,
			},
		},
	}); err != nil {
		t.Fatalf("SaveRecoveries: %v", err)
	}

	if err := db.SaveSleeps([]models.Sleep{
		{
			ID: "11", UserID: 1,
			CreatedAt:  "2024-01-15T00:00:00Z",
			UpdatedAt:  "2024-01-15T07:00:00Z",
			Start:      "2024-01-14T23:00:00Z",
			End:        "2024-01-15T07:00:00Z",
			ScoreState: "SCORED",
			Score: &models.SleepScore{
				StageSummary: models.SleepStageSummary{
					TotalInBedTimeMilli: 28800000,
					TotalAwakeTimeMilli: 3600000,
				},
				SleepEfficiencyPct: 88,
			},
		},
		{
			ID: "12", UserID: 1,
			CreatedAt:  "2024-01-16T00:00:00Z",
			UpdatedAt:  "2024-01-16T07:00:00Z",
			Start:      "2024-01-15T23:00:00Z",
			End:        "2024-01-16T07:00:00Z",
			ScoreState: "SCORED",
			Score: &models.SleepScore{
				StageSummary: models.SleepStageSummary{
					TotalInBedTimeMilli: 25200000,
					TotalAwakeTimeMilli: 1800000,
				},
				SleepEfficiencyPct: 92,
			},
		},
	}); err != nil {
		t.Fatalf("SaveSleeps: %v", err)
	}

	if err := db.SaveCycles([]models.Cycle{
		{
			ID: 1, UserID: 1,
			CreatedAt:  "2024-01-15T00:00:00Z",
			UpdatedAt:  "2024-01-15T00:00:00Z",
			Start:      "2024-01-15T00:00:00Z",
			End:        "2024-01-16T00:00:00Z",
			ScoreState: "SCORED",
			Score:      &models.CycleScore{Strain: 10},
		},
		{
			ID: 2, UserID: 1,
			CreatedAt:  "2024-01-16T00:00:00Z",
			UpdatedAt:  "2024-01-16T00:00:00Z",
			Start:      "2024-01-16T00:00:00Z",
			End:        "2024-01-17T00:00:00Z",
			ScoreState: "SCORED",
			Score:      &models.CycleScore{Strain: 15},
		},
	}); err != nil {
		t.Fatalf("SaveCycles: %v", err)
	}

	if err := db.SaveWorkouts([]models.Workout{
		{
			ID: "100", UserID: 1,
			CreatedAt:  "2024-01-15T09:00:00Z",
			UpdatedAt:  "2024-01-15T10:00:00Z",
			Start:      "2024-01-15T09:00:00Z",
			End:        "2024-01-15T10:00:00Z",
			SportID:    0,
			ScoreState: "SCORED",
			Score:      &models.WorkoutScore{Strain: 8},
		},
		{
			ID: "101", UserID: 1,
			CreatedAt:  "2024-01-16T09:00:00Z",
			UpdatedAt:  "2024-01-16T10:00:00Z",
			Start:      "2024-01-16T09:00:00Z",
			End:        "2024-01-16T10:00:00Z",
			SportID:    1,
			ScoreState: "SCORED",
			Score:      nil,
		},
	}); err != nil {
		t.Fatalf("SaveWorkouts: %v", err)
	}

	weekStart := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	report, err := GenerateWeeklyReport(db, weekStart)
	if err != nil {
		t.Fatalf("GenerateWeeklyReport error = %v", err)
	}

	if report.WeekStart != "2024-01-15" {
		t.Fatalf("WeekStart = %q, want 2024-01-15", report.WeekStart)
	}
	if report.AvgRecovery != 80 {
		t.Fatalf("AvgRecovery = %v, want 80", report.AvgRecovery)
	}
	if report.AvgHRV != 46 {
		t.Fatalf("AvgHRV = %v, want 46", report.AvgHRV)
	}
	if report.AvgRHR != 54 {
		t.Fatalf("AvgRHR = %v, want 54", report.AvgRHR)
	}
	if report.TotalStrain != 25 {
		t.Fatalf("TotalStrain = %v, want 25", report.TotalStrain)
	}
	if report.WorkoutCount != 2 {
		t.Fatalf("WorkoutCount = %d, want 2", report.WorkoutCount)
	}
	if report.AvgWorkoutStrain != 4 {
		t.Fatalf("AvgWorkoutStrain = %v, want 4", report.AvgWorkoutStrain)
	}
}

func TestCheckAlerts(t *testing.T) {
	db := openAnalysisTestDB(t)

	today := time.Now().UTC().Format(time.RFC3339)
	if err := db.SaveRecoveries([]models.Recovery{{
		CycleID: 1, UserID: 1,
		CreatedAt:  today,
		UpdatedAt:  today,
		ScoreState: "SCORED",
		Score:      &models.RecoveryScore{RecoveryScore: 20},
	}}); err != nil {
		t.Fatalf("SaveRecoveries: %v", err)
	}

	if err := db.SaveCycles([]models.Cycle{{
		ID: 1, UserID: 1,
		CreatedAt:  today,
		UpdatedAt:  today,
		Start:      today,
		End:        today,
		ScoreState: "SCORED",
		Score:      &models.CycleScore{Strain: 21},
	}}); err != nil {
		t.Fatalf("SaveCycles: %v", err)
	}

	cfg := &config.Config{Alerts: config.Alerts{Enabled: true, LowRecovery: 33, HighStrain: 18}}
	alerts := CheckAlerts(db, cfg)
	if len(alerts) != 2 {
		t.Fatalf("len(alerts) = %d, want 2", len(alerts))
	}

	disabled := &config.Config{Alerts: config.Alerts{Enabled: false, LowRecovery: 33, HighStrain: 18}}
	if got := CheckAlerts(db, disabled); got != nil {
		t.Fatalf("CheckAlerts with disabled config should return nil")
	}
}
