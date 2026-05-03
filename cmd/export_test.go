package cmd

import (
	"bytes"
	"strings"
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/models"
)

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
				SpO2Percentage:   98,
				SkinTempCelsius:  33.2,
			},
		},
		{CycleID: 2, CreatedAt: "2024-01-16T00:00:00Z"},
	}

	if err := writeCSV(&buf, "recoveries", data); err != nil {
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
				DistanceMeter:    3210,
			},
		},
		{ID: "11", Start: "2024-01-16T10:00:00Z", SportID: 999},
	}

	if err := writeCSV(&buf, "workouts", data); err != nil {
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

	if err := writeCSV(&buf, "sleeps", data); err != nil {
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

	if err := writeCSV(&buf, "cycles", data); err != nil {
		t.Fatalf("writeCSV cycles error = %v", err)
	}

	out := buf.String()
	for _, want := range []string{"id,start,end,strain,kilojoule,avg_hr,max_hr", "30,2024-01-15T00:00:00Z,2024-01-15T06:00:00Z,8.8,700.2,100,155", "31,2024-01-16T00:00:00Z,2024-01-16T06:00:00Z,,,,"} {
		if !strings.Contains(out, want) {
			t.Fatalf("cycles CSV missing %q in:\n%s", want, out)
		}
	}
}
