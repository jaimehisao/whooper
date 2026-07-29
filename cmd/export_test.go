package cmd

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/models"
	"git.infra.hisao.org/hisao/whooper/internal/store"
)

func resetExportFlags() {
	exportFormat = "json"
	exportOutput = ""
	exportEntity = "recoveries"
	exportFrom = ""
	exportTo = ""
	exportCmd.SetOut(nil)
	exportCmd.SetErr(nil)
}

func TestExportCommandDateFiltering(t *testing.T) {
	resetExportFlags()
	defer resetExportFlags()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "whooper.db")
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), dbPath)

	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open DB: %v", err)
	}
	defer db.Close()

	// Seed data for various entities
	if err := db.SaveRecoveries([]models.Recovery{
		{CycleID: 1, UserID: 1, CreatedAt: "2024-01-01T07:00:00Z", ScoreState: "SCORED"},
		{CycleID: 2, UserID: 1, CreatedAt: "2024-01-15T07:00:00Z", ScoreState: "SCORED"},
		{CycleID: 3, UserID: 1, CreatedAt: "2024-02-01T07:00:00Z", ScoreState: "SCORED"},
	}); err != nil {
		t.Fatalf("SaveRecoveries: %v", err)
	}

	if err := db.SaveCycles([]models.Cycle{
		{ID: 10, UserID: 1, Start: "2024-01-01T00:00:00Z", ScoreState: "SCORED"},
		{ID: 11, UserID: 1, Start: "2024-01-15T12:00:00Z", ScoreState: "SCORED"},
		{ID: 12, UserID: 1, Start: "2024-01-15T23:59:59Z", ScoreState: "SCORED"},
		{ID: 13, UserID: 1, Start: "2024-01-16T00:00:01Z", ScoreState: "SCORED"},
	}); err != nil {
		t.Fatalf("SaveCycles: %v", err)
	}

	if err := db.SaveSleeps([]models.Sleep{
		{ID: "20", UserID: 1, Start: "2024-01-01T22:00:00Z", ScoreState: "SCORED"},
		{ID: "21", UserID: 1, Start: "2024-01-15T22:00:00Z", ScoreState: "SCORED"},
		{ID: "22", UserID: 1, Start: "2024-01-16T22:00:00Z", ScoreState: "SCORED"},
	}); err != nil {
		t.Fatalf("SaveSleeps: %v", err)
	}

	if err := db.SaveWorkouts([]models.Workout{
		{ID: "30", UserID: 1, Start: "2024-01-01T10:00:00Z", ScoreState: "SCORED"},
		{ID: "31", UserID: 1, Start: "2024-01-15T10:00:00Z", ScoreState: "SCORED"},
		{ID: "32", UserID: 1, Start: "2024-01-15T20:00:00Z", ScoreState: "SCORED"},
	}); err != nil {
		t.Fatalf("SaveWorkouts: %v", err)
	}

	// Helper to reset global flags to defaults before each subtest
	resetGlobals := resetExportFlags

	t.Run("Recoveries range", func(t *testing.T) {
		resetGlobals()
		exportEntity = "recoveries"
		exportFrom = "2024-01-10"
		exportTo = "2024-01-20"

		var buf bytes.Buffer
		exportCmd.SetOut(&buf)
		if err := exportCmd.RunE(exportCmd, nil); err != nil {
			t.Fatalf("exportCmd.RunE error = %v", err)
		}

		var out []models.Recovery
		if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if len(out) != 1 || out[0].CycleID != 2 {
			t.Errorf("got %d records, want cycle_id 2", len(out))
		}
	})

	t.Run("Cycles inclusive to-day", func(t *testing.T) {
		resetGlobals()
		exportEntity = "cycles"
		exportFrom = "2024-01-15"
		exportTo = "2024-01-15"

		var buf bytes.Buffer
		exportCmd.SetOut(&buf)
		if err := exportCmd.RunE(exportCmd, nil); err != nil {
			t.Fatalf("exportCmd.RunE error = %v", err)
		}

		var out []models.Cycle
		if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		// Should include both ID 11 (noon) and ID 12 (just before midnight)
		if len(out) != 2 {
			t.Errorf("got %d cycles, want 2 (inclusive to-day)", len(out))
		}
	})

	t.Run("Sleeps range", func(t *testing.T) {
		resetGlobals()
		exportEntity = "sleeps"
		exportFrom = "2024-01-01"
		exportTo = "2024-01-15"

		var buf bytes.Buffer
		exportCmd.SetOut(&buf)
		if err := exportCmd.RunE(exportCmd, nil); err != nil {
			t.Fatalf("exportCmd.RunE error = %v", err)
		}

		var out []models.Sleep
		if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if len(out) != 2 {
			t.Errorf("got %d sleeps, want 2", len(out))
		}
	})

	t.Run("Workouts range", func(t *testing.T) {
		resetGlobals()
		exportEntity = "workouts"
		exportFrom = "2024-01-15"
		exportTo = "2024-01-31"

		var buf bytes.Buffer
		exportCmd.SetOut(&buf)
		if err := exportCmd.RunE(exportCmd, nil); err != nil {
			t.Fatalf("exportCmd.RunE error = %v", err)
		}

		var out []models.Workout
		if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if len(out) != 2 {
			t.Errorf("got %d workouts, want 2", len(out))
		}
	})

	t.Run("Invalid from date", func(t *testing.T) {
		resetGlobals()
		exportFrom = "invalid"
		err := exportCmd.RunE(exportCmd, nil)
		if err == nil || !strings.Contains(err.Error(), "invalid 'from' date") {
			t.Fatalf("expected invalid 'from' date error, got %v", err)
		}
	})

	t.Run("To before from", func(t *testing.T) {
		resetGlobals()
		exportFrom = "2024-01-20"
		exportTo = "2024-01-10"
		err := exportCmd.RunE(exportCmd, nil)
		if err == nil || !strings.Contains(err.Error(), "cannot be before") {
			t.Fatalf("expected 'cannot be before' error, got %v", err)
		}
	})

	t.Run("Empty results hint", func(t *testing.T) {
		resetGlobals()
		exportEntity = "recoveries"
		exportFrom = "2025-01-01"
		exportTo = "2025-01-01"

		var stdout, stderr bytes.Buffer
		exportCmd.SetOut(&stdout)
		exportCmd.SetErr(&stderr)

		if err := exportCmd.RunE(exportCmd, nil); err != nil {
			t.Fatalf("exportCmd.RunE error = %v", err)
		}

		if !strings.Contains(stdout.String(), "[]") {
			t.Errorf("expected empty JSON array on stdout, got %q", stdout.String())
		}
		if !strings.Contains(stderr.String(), "No recoveries found") {
			t.Errorf("expected hint on stderr, got %q", stderr.String())
		}
	})
}

func TestWriteCSVMaps(t *testing.T) {
	var buf bytes.Buffer
	rows := []map[string]any{
		{"day": "2024-06-01", "recovery_score": 88.0},
		{"day": "2024-06-02", "recovery_score": 70.0},
	}
	if err := writeCSVMaps(&buf, rows); err != nil {
		t.Fatalf("writeCSVMaps: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "day") || !strings.Contains(out, "recovery_score") {
		t.Fatalf("missing headers: %s", out)
	}
	if !strings.Contains(out, "2024-06-01") || !strings.Contains(out, "88") {
		t.Fatalf("missing values: %s", out)
	}
	var empty bytes.Buffer
	if err := writeCSVMaps(&empty, nil); err != nil {
		t.Fatalf("empty maps: %v", err)
	}
}

func TestWriteCSVRecoveries(t *testing.T) {
	var buf bytes.Buffer
	data := []models.Recovery{
		{
			CycleID:   1,
			CreatedAt: "2024-01-15T00:00:00Z",
			Score: &models.RecoveryScore{
				RecoveryScore:    75,
				HRVRmssd:         45,
				RestingHeartRate: 55,
				SpO2Percentage: models.FloatPtr(98),
				SkinTempCelsius: models.FloatPtr(33.2),
			},
		},
		{CycleID: 2, CreatedAt: "2024-01-16T00:00:00Z"},
	}

	if err := writeCSVData(&buf, "recoveries", data); err != nil {
		t.Fatalf("writeCSV recoveries error = %v", err)
	}

	out := buf.String()
	for _, want := range []string{"cycle_id,date,recovery_score,hrv,rhr,spo2,skin_temp", "1,2024-01-15T00:00:00Z,75.0,45.0,55.0,98.0,33.2", "2,2024-01-16T00:00:00Z,,,,,"} {
		if !strings.Contains(out, want) {
			t.Fatalf("recoveries CSV missing %q in:\n%s", want, out)
		}
	}
}

func TestWriteCSVWorkouts(t *testing.T) {
	var buf bytes.Buffer
	data := []models.Workout{
		{
			ID:      "10",
			Start:   "2024-01-15T10:00:00Z",
			SportID: 1,
			Score: &models.WorkoutScore{
				Strain:           12.3,
				AverageHeartRate: 140,
				MaxHeartRate:     180,
				Kilojoule:        500.5,
				DistanceMeter: models.FloatPtr(3210),
			},
		},
		{ID: "11", Start: "2024-01-16T10:00:00Z", SportID: 999},
	}

	if err := writeCSVData(&buf, "workouts", data); err != nil {
		t.Fatalf("writeCSV workouts error = %v", err)
	}

	out := buf.String()
	for _, want := range []string{"id,date,sport_id,sport,strain,avg_hr,max_hr,kilojoule,distance_m", "10,2024-01-15T10:00:00Z,1,Cycling,12.3,140,180,500.5,3210.0", "11,2024-01-16T10:00:00Z,999,,,,,,"} {
		if !strings.Contains(out, want) {
			t.Fatalf("workouts CSV missing %q in:\n%s", want, out)
		}
	}
}

func TestWriteCSVSleeps(t *testing.T) {
	var buf bytes.Buffer
	data := []models.Sleep{
		{
			ID:    "20",
			Start: "2024-01-15T22:00:00Z",
			End:   "2024-01-16T06:00:00Z",
			Nap:   true,
			Score: &models.SleepScore{
				SleepPerformancePct: 90,
				SleepEfficiencyPct:  88.5,
				RespiratoryRate:     16.2,
			},
		},
		{ID: "21", Start: "2024-01-16T22:00:00Z", End: "2024-01-17T06:00:00Z"},
	}

	if err := writeCSVData(&buf, "sleeps", data); err != nil {
		t.Fatalf("writeCSV sleeps error = %v", err)
	}

	out := buf.String()
	for _, want := range []string{"id,start,end,nap,performance_pct,efficiency_pct,respiratory_rate", "20,2024-01-15T22:00:00Z,2024-01-16T06:00:00Z,true,90.0,88.5,16.2", "21,2024-01-16T22:00:00Z,2024-01-17T06:00:00Z,false,,,"} {
		if !strings.Contains(out, want) {
			t.Fatalf("sleeps CSV missing %q in:\n%s", want, out)
		}
	}
}

func TestWriteCSVCycles(t *testing.T) {
	var buf bytes.Buffer
	data := []models.Cycle{
		{
			ID:    30,
			Start: "2024-01-15T00:00:00Z",
			End:   "2024-01-15T06:00:00Z",
			Score: &models.CycleScore{
				Strain:           8.8,
				Kilojoule:        700.2,
				AverageHeartRate: 100,
				MaxHeartRate:     155,
			},
		},
		{ID: 31, Start: "2024-01-16T00:00:00Z", End: "2024-01-16T06:00:00Z"},
	}

	if err := writeCSVData(&buf, "cycles", data); err != nil {
		t.Fatalf("writeCSV cycles error = %v", err)
	}

	out := buf.String()
	for _, want := range []string{"id,start,end,strain,kilojoule,avg_hr,max_hr", "30,2024-01-15T00:00:00Z,2024-01-15T06:00:00Z,8.8,700.2,100,155", "31,2024-01-16T00:00:00Z,2024-01-16T06:00:00Z,,,,"} {
		if !strings.Contains(out, want) {
			t.Fatalf("cycles CSV missing %q in:\n%s", want, out)
		}
	}
}
