package store

import (
	"os"
	"path/filepath"
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/models"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOpen_CreatesTables(t *testing.T) {
	db := openTestDB(t)

	tables := []string{"profile", "cycle", "recovery", "sleep", "workout", "sync_state"}
	for _, table := range tables {
		var name string
		err := db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestOpenReadOnly(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := db.SaveProfile(models.Profile{UserID: 123, Email: "test@example.com", FirstName: "Jane", LastName: "Doe"}); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close writable DB: %v", err)
	}

	readOnly, err := OpenReadOnly(dbPath)
	if err != nil {
		t.Fatalf("OpenReadOnly: %v", err)
	}
	defer readOnly.Close()

	profile, err := readOnly.GetProfile()
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if profile.UserID != 123 {
		t.Fatalf("read-only profile user ID = %d, want 123", profile.UserID)
	}
	if err := readOnly.SaveProfile(models.Profile{UserID: 456, Email: "other@example.com"}); err == nil {
		t.Fatal("expected write through read-only DB to fail")
	}
}

func TestOpenReadOnlyMissingDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "missing.db")
	if _, err := OpenReadOnly(dbPath); err == nil {
		t.Fatal("expected OpenReadOnly to fail for missing database")
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("OpenReadOnly should not create database, stat err = %v", err)
	}
}

func TestOpenCreatesParentDirectory(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "whooper.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(filepath.Dir(dbPath)); err != nil {
		t.Fatalf("expected parent directory to exist: %v", err)
	}
}

func TestSaveProfile_GetProfile(t *testing.T) {
	db := openTestDB(t)

	p := models.Profile{
		UserID:    12345,
		Email:     "test@example.com",
		FirstName: "Jane",
		LastName:  "Doe",
	}
	if err := db.SaveProfile(p); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	got, err := db.GetProfile()
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if got.UserID != p.UserID || got.Email != p.Email ||
		got.FirstName != p.FirstName || got.LastName != p.LastName {
		t.Errorf("profile mismatch: got %+v, want %+v", got, p)
	}
}

func TestSaveCycles_ListCycles(t *testing.T) {
	db := openTestDB(t)

	cycles := []models.Cycle{
		{
			ID: 1, UserID: 100,
			CreatedAt: "2025-01-15T08:00:00Z", UpdatedAt: "2025-01-15T08:00:00Z",
			Start: "2025-01-15T00:00:00Z", End: "2025-01-16T00:00:00Z",
			Days: 1, ScoreState: "SCORED",
			Score: &models.CycleScore{Strain: 12.5, Kilojoule: 8500, AverageHeartRate: 72, MaxHeartRate: 155},
		},
		{
			ID: 2, UserID: 100,
			CreatedAt: "2025-01-16T08:00:00Z", UpdatedAt: "2025-01-16T08:00:00Z",
			Start: "2025-01-16T00:00:00Z", End: "2025-01-17T00:00:00Z",
			Days: 1, ScoreState: "SCORED",
			Score: &models.CycleScore{Strain: 14.2, Kilojoule: 9200, AverageHeartRate: 75, MaxHeartRate: 168},
		},
	}

	if err := db.SaveCycles(cycles); err != nil {
		t.Fatalf("SaveCycles: %v", err)
	}

	// List all
	all, err := db.ListCycles("", "")
	if err != nil {
		t.Fatalf("ListCycles all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 cycles, got %d", len(all))
	}

	// Filter by date
	filtered, err := db.ListCycles("2025-01-16T00:00:00Z", "")
	if err != nil {
		t.Fatalf("ListCycles filtered: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 cycle after filter, got %d", len(filtered))
	}
	if filtered[0].ID != 2 {
		t.Errorf("expected cycle ID 2, got %d", filtered[0].ID)
	}
}

func TestSaveRecoveries_ListRecoveries(t *testing.T) {
	db := openTestDB(t)

	recoveries := []models.Recovery{
		{
			CycleID: 1, SleepID: "10", UserID: 100,
			CreatedAt: "2025-01-15T08:00:00Z", UpdatedAt: "2025-01-15T08:00:00Z",
			ScoreState: "SCORED",
			Score: &models.RecoveryScore{
				UserCalibrating:  false,
				RecoveryScore:    78.5,
				RestingHeartRate: 52.0,
				HRVRmssd:         45.2,
				SpO2Percentage:   97.0,
				SkinTempCelsius:  33.5,
			},
		},
		{
			CycleID: 2, SleepID: "11", UserID: 100,
			CreatedAt: "2025-01-16T08:00:00Z", UpdatedAt: "2025-01-16T08:00:00Z",
			ScoreState: "SCORED",
			Score: &models.RecoveryScore{
				UserCalibrating:  false,
				RecoveryScore:    65.0,
				RestingHeartRate: 55.0,
				HRVRmssd:         38.7,
				SpO2Percentage:   96.0,
				SkinTempCelsius:  33.8,
			},
		},
	}

	if err := db.SaveRecoveries(recoveries); err != nil {
		t.Fatalf("SaveRecoveries: %v", err)
	}

	all, err := db.ListRecoveries("", "")
	if err != nil {
		t.Fatalf("ListRecoveries: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 recoveries, got %d", len(all))
	}
	if all[0].Score.RecoveryScore != 65.0 {
		t.Errorf("expected first recovery score 65.0 (DESC order), got %f", all[0].Score.RecoveryScore)
	}
}

func TestSaveSleeps_ListSleeps_ExcludeNaps(t *testing.T) {
	db := openTestDB(t)

	sleeps := []models.Sleep{
		{
			ID: "10", UserID: 100,
			CreatedAt: "2025-01-15T06:00:00Z", UpdatedAt: "2025-01-15T06:00:00Z",
			Start: "2025-01-14T22:00:00Z", End: "2025-01-15T06:00:00Z",
			Nap: false, ScoreState: "SCORED",
			Score: &models.SleepScore{
				StageSummary: models.SleepStageSummary{
					TotalInBedTimeMilli:         28800000,
					TotalAwakeTimeMilli:         3600000,
					TotalNoDataTimeMilli:        0,
					TotalLightSleepTimeMilli:    10800000,
					TotalSlowWaveSleepTimeMilli: 7200000,
					TotalRemSleepTimeMilli:      7200000,
					SleepCycleCount:             4,
					DisturbanceCount:            2,
				},
				SleepNeeded: models.SleepNeeded{
					BaselineMilli:             28800000,
					NeedFromSleepDebtMilli:    1800000,
					NeedFromRecentStrainMilli: 900000,
					NeedFromRecentNapMilli:    0,
				},
				RespiratoryRate:     15.2,
				SleepPerformancePct: 92.0,
				SleepConsistencyPct: 85.0,
				SleepEfficiencyPct:  87.5,
			},
		},
		{
			ID: "11", UserID: 100,
			CreatedAt: "2025-01-15T15:00:00Z", UpdatedAt: "2025-01-15T15:00:00Z",
			Start: "2025-01-15T13:00:00Z", End: "2025-01-15T13:30:00Z",
			Nap: true, ScoreState: "SCORED",
			Score: &models.SleepScore{
				StageSummary: models.SleepStageSummary{
					TotalInBedTimeMilli: 1800000,
					TotalAwakeTimeMilli: 300000,
				},
				RespiratoryRate:     14.8,
				SleepPerformancePct: 0,
				SleepConsistencyPct: 0,
				SleepEfficiencyPct:  83.3,
			},
		},
	}

	if err := db.SaveSleeps(sleeps); err != nil {
		t.Fatalf("SaveSleeps: %v", err)
	}

	// All sleeps
	all, err := db.ListSleeps("", "", false)
	if err != nil {
		t.Fatalf("ListSleeps all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 sleeps, got %d", len(all))
	}

	// Exclude naps
	noNaps, err := db.ListSleeps("", "", true)
	if err != nil {
		t.Fatalf("ListSleeps excludeNaps: %v", err)
	}
	if len(noNaps) != 1 {
		t.Fatalf("expected 1 sleep excluding naps, got %d", len(noNaps))
	}
	if noNaps[0].ID != "10" {
		t.Errorf("expected sleep ID 10, got %s", noNaps[0].ID)
	}
}

func TestSaveWorkouts_ListWorkouts(t *testing.T) {
	db := openTestDB(t)

	workouts := []models.Workout{
		{
			ID: "500", UserID: 100,
			CreatedAt: "2025-01-15T10:00:00Z", UpdatedAt: "2025-01-15T10:00:00Z",
			Start: "2025-01-15T09:00:00Z", End: "2025-01-15T10:00:00Z",
			SportID: 0, ScoreState: "SCORED",
			Score: &models.WorkoutScore{
				Strain:              8.5,
				AverageHeartRate:    145,
				MaxHeartRate:        175,
				Kilojoule:           1200,
				PercentRecorded:     100.0,
				DistanceMeter:       5000,
				AltitudeGainMeter:   50.0,
				AltitudeChangeMeter: 10.0,
				ZoneDuration: &models.ZoneDuration{
					ZoneZeroMilli:  60000,
					ZoneOneMilli:   300000,
					ZoneTwoMilli:   600000,
					ZoneThreeMilli: 1200000,
					ZoneFourMilli:  900000,
					ZoneFiveMilli:  540000,
				},
			},
		},
	}

	if err := db.SaveWorkouts(workouts); err != nil {
		t.Fatalf("SaveWorkouts: %v", err)
	}

	all, err := db.ListWorkouts("", "")
	if err != nil {
		t.Fatalf("ListWorkouts: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 workout, got %d", len(all))
	}
	w := all[0]
	if w.ID != "500" {
		t.Errorf("expected workout ID 500, got %s", w.ID)
	}
	if w.Score == nil {
		t.Fatal("expected non-nil score")
	}
	if w.Score.Strain != 8.5 {
		t.Errorf("expected strain 8.5, got %f", w.Score.Strain)
	}
	if w.Score.ZoneDuration == nil {
		t.Fatal("expected non-nil zone duration")
	}
	if w.Score.ZoneDuration.ZoneThreeMilli != 1200000 {
		t.Errorf("expected zone three 1200000, got %d", w.Score.ZoneDuration.ZoneThreeMilli)
	}
}

func TestGetSyncState_SetSyncState(t *testing.T) {
	db := openTestDB(t)

	// Initially empty
	val, err := db.GetSyncState("cycles")
	if err != nil {
		t.Fatalf("GetSyncState: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty string for missing key, got %q", val)
	}

	// Set and get
	if err := db.SetSyncState("cycles", "2025-01-15T00:00:00Z"); err != nil {
		t.Fatalf("SetSyncState: %v", err)
	}
	val, err = db.GetSyncState("cycles")
	if err != nil {
		t.Fatalf("GetSyncState after set: %v", err)
	}
	if val != "2025-01-15T00:00:00Z" {
		t.Errorf("expected 2025-01-15T00:00:00Z, got %q", val)
	}

	// Update
	if err := db.SetSyncState("cycles", "2025-01-16T00:00:00Z"); err != nil {
		t.Fatalf("SetSyncState update: %v", err)
	}
	val, err = db.GetSyncState("cycles")
	if err != nil {
		t.Fatalf("GetSyncState after update: %v", err)
	}
	if val != "2025-01-16T00:00:00Z" {
		t.Errorf("expected 2025-01-16T00:00:00Z, got %q", val)
	}
}

func TestGetRecoveryTrend_DateOnlyToIncludesLastDay(t *testing.T) {
	db := openTestDB(t)
	if err := db.SaveRecoveries([]models.Recovery{
		{
			CycleID: 1, SleepID: "10", UserID: 100,
			CreatedAt: "2024-01-08T07:00:00Z", UpdatedAt: "2024-01-08T07:00:00Z",
			ScoreState: "SCORED",
			Score:      &models.RecoveryScore{RecoveryScore: 70, RestingHeartRate: 50, HRVRmssd: 40},
		},
	}); err != nil {
		t.Fatalf("SaveRecoveries: %v", err)
	}

	points, err := db.GetRecoveryTrend("2024-01-08", "2024-01-08")
	if err != nil {
		t.Fatalf("GetRecoveryTrend: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("len(points) = %d, want 1 (date-only to must include mid-day timestamps)", len(points))
	}
}

func TestGetRecoveryTrend_DeduplicatesSameDay(t *testing.T) {
	db := openTestDB(t)
	if err := db.SaveRecoveries([]models.Recovery{
		{
			CycleID: 1, SleepID: "10", UserID: 100,
			CreatedAt: "2024-01-08T06:00:00Z", UpdatedAt: "2024-01-08T06:00:00Z",
			ScoreState: "SCORED",
			Score:      &models.RecoveryScore{RecoveryScore: 20, RestingHeartRate: 50, HRVRmssd: 30},
		},
		{
			CycleID: 2, SleepID: "11", UserID: 100,
			CreatedAt: "2024-01-08T12:00:00Z", UpdatedAt: "2024-01-08T12:00:00Z",
			ScoreState: "SCORED",
			Score:      &models.RecoveryScore{RecoveryScore: 80, RestingHeartRate: 48, HRVRmssd: 55},
		},
	}); err != nil {
		t.Fatalf("SaveRecoveries: %v", err)
	}

	points, err := db.GetRecoveryTrend("2024-01-08", "2024-01-08")
	if err != nil {
		t.Fatalf("GetRecoveryTrend: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("len(points) = %d, want 1 (one row per day)", len(points))
	}
	if points[0].RecoveryScore != 80 {
		t.Fatalf("RecoveryScore = %v, want latest (80)", points[0].RecoveryScore)
	}
}

func TestGetRecoveryTrend(t *testing.T) {
	db := openTestDB(t)

	recoveries := []models.Recovery{
		{
			CycleID: 1, SleepID: "10", UserID: 100,
			CreatedAt: "2025-01-15T08:00:00Z", UpdatedAt: "2025-01-15T08:00:00Z",
			ScoreState: "SCORED",
			Score: &models.RecoveryScore{
				RecoveryScore: 78.5, RestingHeartRate: 52.0, HRVRmssd: 45.2,
			},
		},
		{
			CycleID: 2, SleepID: "11", UserID: 100,
			CreatedAt: "2025-01-16T08:00:00Z", UpdatedAt: "2025-01-16T08:00:00Z",
			ScoreState: "SCORED",
			Score: &models.RecoveryScore{
				RecoveryScore: 65.0, RestingHeartRate: 55.0, HRVRmssd: 38.7,
			},
		},
		{
			CycleID: 3, SleepID: "12", UserID: 100,
			CreatedAt: "2025-01-17T08:00:00Z", UpdatedAt: "2025-01-17T08:00:00Z",
			ScoreState: "PENDING",
			Score:      nil,
		},
	}

	if err := db.SaveRecoveries(recoveries); err != nil {
		t.Fatalf("SaveRecoveries: %v", err)
	}

	points, err := db.GetRecoveryTrend("", "")
	if err != nil {
		t.Fatalf("GetRecoveryTrend: %v", err)
	}
	// Only SCORED records
	if len(points) != 2 {
		t.Fatalf("expected 2 trend points, got %d", len(points))
	}
	if points[0].RecoveryScore != 78.5 {
		t.Errorf("expected first recovery score 78.5, got %f", points[0].RecoveryScore)
	}
	if points[1].HRV != 38.7 {
		t.Errorf("expected second HRV 38.7, got %f", points[1].HRV)
	}
}

func TestGetCorrelationData(t *testing.T) {
	db := openTestDB(t)

	// Insert recoveries with SCORED state
	recoveries := []models.Recovery{
		{
			CycleID: 1, SleepID: "10", UserID: 100,
			CreatedAt: "2025-01-15T08:00:00Z", UpdatedAt: "2025-01-15T08:00:00Z",
			ScoreState: "SCORED",
			Score: &models.RecoveryScore{
				RecoveryScore: 78.5, HRVRmssd: 45.2,
			},
		},
		{
			CycleID: 2, SleepID: "11", UserID: 100,
			CreatedAt: "2025-01-16T08:00:00Z", UpdatedAt: "2025-01-16T08:00:00Z",
			ScoreState: "SCORED",
			Score: &models.RecoveryScore{
				RecoveryScore: 65.0, HRVRmssd: 38.7,
			},
		},
	}

	if err := db.SaveRecoveries(recoveries); err != nil {
		t.Fatalf("SaveRecoveries: %v", err)
	}

	// Both metrics from same table
	points, err := db.GetCorrelationData("recovery", "hrv")
	if err != nil {
		t.Fatalf("GetCorrelationData: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("expected 2 correlation points, got %d", len(points))
	}
	if points[0].X != 78.5 || points[0].Y != 45.2 {
		t.Errorf("unexpected first point: %+v", points[0])
	}
}
