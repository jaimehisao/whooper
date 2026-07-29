package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/models"
	"git.infra.hisao.org/hisao/whooper/internal/store"
)

func setupSummaryEnv(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), filepath.Join(tmpDir, "whooper.db"))
	return tmpDir
}

func TestSummaryEmpty(t *testing.T) {
	_ = setupSummaryEnv(t)
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatalf("Open DB: %v", err)
	}
	db.Close()

	var buf bytes.Buffer
	summaryCmd.SetOut(&buf)
	defer summaryCmd.SetOut(nil)
	if err := summaryCmd.RunE(summaryCmd, []string{}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No local health data found.") {
		t.Errorf("expected no data message, got:\n%s", output)
	}
	if !strings.Contains(output, "Hint: Run 'whooper sync'") {
		t.Errorf("expected hint, got:\n%s", output)
	}
}

func TestSummaryPopulated(t *testing.T) {
	_ = setupSummaryEnv(t)
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatalf("Open DB: %v", err)
	}

	// Add some data
	recovery := models.Recovery{
		CycleID:    1,
		CreatedAt:  "2024-05-19T10:00:00Z",
		ScoreState: "SCORED",
		Score: &models.RecoveryScore{
			RecoveryScore:    75,
			HRVRmssd:         65,
			RestingHeartRate: 50,
		},
	}
	if err := db.SaveRecoveries([]models.Recovery{recovery}); err != nil {
		t.Fatalf("SaveRecoveries: %v", err)
	}

	sleep := models.Sleep{
		ID:         "1",
		Start:      "2024-05-19T00:00:00Z",
		End:        "2024-05-19T07:30:00Z",
		Nap:        false,
		ScoreState: "SCORED",
		Score: &models.SleepScore{
			StageSummary: models.SleepStageSummary{
				TotalInBedTimeMilli: 28800000, // 8h
				TotalAwakeTimeMilli: 1800000,  // 0.5h
			},
			SleepNeeded: models.SleepNeeded{
				BaselineMilli:          28800000,
				NeedFromSleepDebtMilli: 3600000,
			},
			SleepEfficiencyPct:  94,
			SleepPerformancePct: 85,
		},
	}
	if err := db.SaveSleeps([]models.Sleep{sleep}); err != nil {
		t.Fatalf("SaveSleeps: %v", err)
	}

	workout := models.Workout{
		ID:         "w1",
		Start:      "2024-05-19T17:00:00Z",
		End:        "2024-05-19T18:00:00Z",
		SportID:    1, // Cycling
		ScoreState: "SCORED",
		Score: &models.WorkoutScore{
			Strain:           12.5,
			AverageHeartRate: 145,
			MaxHeartRate:     170,
			DistanceMeter: models.FloatPtr(25000),
		},
	}
	if err := db.SaveWorkouts([]models.Workout{workout}); err != nil {
		t.Fatalf("SaveWorkouts: %v", err)
	}

	if err := db.SetSyncState("cycles", "2024-05-19T20:00:00Z"); err != nil {
		t.Fatalf("SetSyncState: %v", err)
	}

	db.Close()

	var buf bytes.Buffer
	summaryCmd.SetOut(&buf)
	defer summaryCmd.SetOut(nil)
	if err := summaryCmd.RunE(summaryCmd, []string{}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	output := buf.String()
	expectedSubstrings := []string{
		"Recovery: 75%",
		"HRV:      65 ms",
		"Sleep:    7.5h",
		"Debt: 1.0h",
		"Cycling: 12.5 strain",
		"60 min",
		"25.00 km",
		"cycles:      2024-05-19T20:00:00Z",
	}
	for _, s := range expectedSubstrings {
		if !strings.Contains(output, s) {
			t.Errorf("output missing %q:\n%s", s, output)
		}
	}
}

func TestSummaryJSON(t *testing.T) {
	_ = setupSummaryEnv(t)
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatalf("Open DB: %v", err)
	}
	db.Close()

	var buf bytes.Buffer
	summaryCmd.SetOut(&buf)
	defer summaryCmd.SetOut(nil)

	summaryJSON = true
	if err := summaryCmd.RunE(summaryCmd, []string{}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	var report struct {
		LatestHealth *healthReport     `json:"latest_health"`
		LastSync     map[string]string `json:"last_sync"`
	}
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("Unmarshal JSON: %v\n%s", err, buf.String())
	}
}

func TestSummaryDBOpenError(t *testing.T) {
	tmpDir := t.TempDir()
	// Use a path that will fail to open as a DB (file where a directory should be)
	blockedDir := filepath.Join(tmpDir, "blocked")
	if err := os.WriteFile(blockedDir, []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(blockedDir, "whooper.db")
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), dbPath)

	err := summaryCmd.RunE(summaryCmd, []string{})
	if err == nil {
		t.Fatal("expected error for failed db open")
	}
	if !strings.Contains(err.Error(), "open database") {
		t.Fatalf("unexpected error: %v", err)
	}
}
