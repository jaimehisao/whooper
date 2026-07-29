package store

import (
	"path/filepath"
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/models"
)

func TestDB_Open(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open error = %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Ping error = %v", err)
	}
}

func TestDB_SaveAndGetProfile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open error = %v", err)
	}
	defer db.Close()

	profile := models.Profile{
		UserID:    123,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
	}

	if err := db.SaveProfile(profile); err != nil {
		t.Fatalf("SaveProfile error = %v", err)
	}

	got, err := db.GetProfile()
	if err != nil {
		t.Fatalf("GetProfile error = %v", err)
	}
	if got.UserID != 123 {
		t.Errorf("UserID = %d, want 123", got.UserID)
	}
	if got.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", got.Email)
	}
}

func TestDB_SaveCycles(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open error = %v", err)
	}
	defer db.Close()

	cycles := []models.Cycle{
		{
			ID:         1,
			UserID:     123,
			Start:      "2024-01-15T00:00:00Z",
			End:        "2024-01-15T06:00:00Z",
			ScoreState: "SCORED",
			Score: &models.CycleScore{
				Strain:           12.5,
				Kilojoule:        1200.5,
				AverageHeartRate: 120,
				MaxHeartRate:     180,
			},
		},
	}

	if err := db.SaveCycles(cycles); err != nil {
		t.Fatalf("SaveCycles error = %v", err)
	}

	got, err := db.ListCycles("2024-01-01", "2024-12-31")
	if err != nil {
		t.Fatalf("ListCycles error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].Score == nil || got[0].Score.Strain != 12.5 {
		t.Errorf("Strain = %v, want 12.5", got[0].Score.Strain)
	}
}

func TestDB_SaveRecoveries(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open error = %v", err)
	}
	defer db.Close()

	recoveries := []models.Recovery{
		{
			CycleID:    1,
			SleepID:    "1",
			UserID:     123,
			CreatedAt:  "2024-01-15T00:00:00Z",
			UpdatedAt:  "2024-01-15T06:00:00Z",
			ScoreState: "SCORED",
			Score: &models.RecoveryScore{
				UserCalibrating:  false,
				RecoveryScore:    75,
				RestingHeartRate: 55,
				HRVRmssd:         45.5,
				SpO2Percentage: models.FloatPtr(98.5),
				SkinTempCelsius: models.FloatPtr(33.2),
			},
		},
	}

	if err := db.SaveRecoveries(recoveries); err != nil {
		t.Fatalf("SaveRecoveries error = %v", err)
	}

	got, err := db.ListRecoveries("2024-01-01", "2024-12-31")
	if err != nil {
		t.Fatalf("ListRecoveries error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].Score == nil || got[0].Score.RecoveryScore != 75 {
		t.Errorf("RecoveryScore = %v, want 75", got[0].Score.RecoveryScore)
	}
}

func TestDB_SaveSleeps(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open error = %v", err)
	}
	defer db.Close()

	sleeps := []models.Sleep{
		{
			ID:         "1",
			UserID:     123,
			Start:      "2024-01-14T22:00:00Z",
			End:        "2024-01-15T06:00:00Z",
			Nap:        false,
			ScoreState: "SCORED",
			Score: &models.SleepScore{
				StageSummary: models.SleepStageSummary{
					TotalInBedTimeMilli:      28800000,
					TotalLightSleepTimeMilli: 18000000,
					TotalAwakeTimeMilli:      1800000,
				},
				SleepEfficiencyPct:  92.5,
				SleepPerformancePct: 95.0,
			},
		},
	}

	if err := db.SaveSleeps(sleeps); err != nil {
		t.Fatalf("SaveSleeps error = %v", err)
	}

	got, err := db.ListSleeps("2024-01-01", "2024-12-31", false)
	if err != nil {
		t.Fatalf("ListSleeps error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].Score == nil || got[0].Score.SleepEfficiencyPct != 92.5 {
		t.Errorf("SleepEfficiencyPct = %v, want 92.5", got[0].Score.SleepEfficiencyPct)
	}
}

func TestDB_SaveWorkouts(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open error = %v", err)
	}
	defer db.Close()

	workouts := []models.Workout{
		{
			ID:         "1",
			UserID:     123,
			Start:      "2024-01-15T10:00:00Z",
			End:        "2024-01-15T11:00:00Z",
			SportID:    1,
			ScoreState: "SCORED",
			Score: &models.WorkoutScore{
				Strain:           15.2,
				AverageHeartRate: 145,
				MaxHeartRate:     185,
				Kilojoule:        800.0,
				DistanceMeter: models.FloatPtr(5000),
			},
		},
	}

	if err := db.SaveWorkouts(workouts); err != nil {
		t.Fatalf("SaveWorkouts error = %v", err)
	}

	got, err := db.ListWorkouts("2024-01-01", "2024-12-31")
	if err != nil {
		t.Fatalf("ListWorkouts error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].Score == nil || got[0].Score.Strain != 15.2 {
		t.Errorf("Strain = %v, want 15.2", got[0].Score.Strain)
	}
}

func TestDB_GetSyncState(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open error = %v", err)
	}
	defer db.Close()

	got, err := db.GetSyncState("cycles")
	if err != nil {
		t.Fatalf("GetSyncState error = %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string for new db, got %q", got)
	}

	if err := db.SetSyncState("cycles", "2024-01-15T00:00:00Z"); err != nil {
		t.Fatalf("SetSyncState error = %v", err)
	}

	got, err = db.GetSyncState("cycles")
	if err != nil {
		t.Fatalf("GetSyncState error = %v", err)
	}
	if got != "2024-01-15T00:00:00Z" {
		t.Errorf("GetSyncState = %q, want 2024-01-15T00:00:00Z", got)
	}
}

func TestDB_Trends(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open error = %v", err)
	}
	defer db.Close()

	recoveries := []models.Recovery{
		{
			CycleID:    1,
			UserID:     123,
			CreatedAt:  "2024-01-15T00:00:00Z",
			ScoreState: "SCORED",
			Score: &models.RecoveryScore{
				RecoveryScore:    75,
				HRVRmssd:         45.0,
				RestingHeartRate: 55,
			},
		},
	}
	if err := db.SaveRecoveries(recoveries); err != nil {
		t.Fatalf("SaveRecoveries error = %v", err)
	}

	cycles := []models.Cycle{
		{
			ID:         1,
			UserID:     123,
			Start:      "2024-01-15T00:00:00Z",
			End:        "2024-01-15T06:00:00Z",
			ScoreState: "SCORED",
			Score: &models.CycleScore{
				Strain: 12.5,
			},
		},
	}
	if err := db.SaveCycles(cycles); err != nil {
		t.Fatalf("SaveCycles error = %v", err)
	}

	recoveryTrends, err := db.GetRecoveryTrend("2024-01-01", "2024-12-31")
	if err != nil {
		t.Fatalf("GetRecoveryTrend error = %v", err)
	}
	if len(recoveryTrends) != 1 {
		t.Fatalf("len(recoveryTrends) = %d, want 1", len(recoveryTrends))
	}

	strainTrends, err := db.GetStrainTrend("2024-01-01", "2024-12-31")
	if err != nil {
		t.Fatalf("GetStrainTrend error = %v", err)
	}
	if len(strainTrends) != 1 {
		t.Fatalf("len(strainTrends) = %d, want 1", len(strainTrends))
	}
}

func TestDB_Correlations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open error = %v", err)
	}
	defer db.Close()

	recoveries := []models.Recovery{
		{
			CycleID:    1,
			UserID:     123,
			CreatedAt:  "2024-01-15T00:00:00Z",
			ScoreState: "SCORED",
			Score: &models.RecoveryScore{
				RecoveryScore:    75,
				HRVRmssd:         45.0,
				RestingHeartRate: 55,
			},
		},
	}
	if err := db.SaveRecoveries(recoveries); err != nil {
		t.Fatalf("SaveRecoveries error = %v", err)
	}

	points, err := db.GetCorrelationDataSince("recovery", "hrv", 0)
	if err != nil {
		t.Fatalf("GetCorrelationData error = %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("len(points) = %d, want 1", len(points))
	}
	if points[0].X != 75 || points[0].Y != 45.0 {
		t.Errorf("points[0] = (%v, %v), want (75, 45.0)", points[0].X, points[0].Y)
	}
}

func TestDB_OpenInvalidPath(t *testing.T) {
	_, err := Open("/nonexistent/path/test.db")
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}
