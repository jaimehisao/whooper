package store

import (
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/models"
)

func TestViewShapes(t *testing.T) {
	db := openTestDB(t)

	// Seed minimal data to exercise computed columns
	err := db.SaveSleeps([]models.Sleep{
		{
			ID: "v1", UserID: 1, Start: "2024-01-01T00:00:00Z", End: "2024-01-01T08:00:00Z",
			Nap: false, ScoreState: "SCORED",
			Score: &models.SleepScore{
				StageSummary: models.SleepStageSummary{
					TotalInBedTimeMilli: 28800000,
					TotalAwakeTimeMilli: 3600000,
				},
				SleepNeeded: models.SleepNeeded{
					BaselineMilli: 28800000,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("SaveSleeps: %v", err)
	}

	err = db.SaveWorkouts([]models.Workout{
		{
			ID: "w1", UserID: 1, Start: "2024-01-01T10:00:00Z", End: "2024-01-01T11:00:00Z",
			ScoreState: "SCORED",
			Score: &models.WorkoutScore{
				Strain:        10.5,
				DistanceMeter: 5000,
			},
		},
	})
	if err != nil {
		t.Fatalf("SaveWorkouts: %v", err)
	}

	t.Run("daily_recovery_shape", func(t *testing.T) {
		rows, err := db.Query(`SELECT day, recovery_score, hrv_rmssd, resting_heart_rate FROM daily_recovery`)
		if err != nil {
			t.Fatalf("query daily_recovery: %v", err)
		}
		defer rows.Close()
		cols, _ := rows.Columns()
		expected := []string{"day", "recovery_score", "hrv_rmssd", "resting_heart_rate"}
		for i, name := range expected {
			if cols[i] != name {
				t.Errorf("col[%d] = %s, want %s", i, cols[i], name)
			}
		}
	})

	t.Run("daily_sleep_computations", func(t *testing.T) {
		var day string
		var actualHours, needHours float64
		err := db.QueryRow(`SELECT day, actual_hours, need_hours FROM daily_sleep`).Scan(&day, &actualHours, &needHours)
		if err != nil {
			t.Fatalf("query daily_sleep: %v", err)
		}
		if day != "2024-01-01" {
			t.Errorf("day = %s, want 2024-01-01", day)
		}
		// (28800000 - 3600000) / 3600000 = 7.0
		if actualHours != 7.0 {
			t.Errorf("actualHours = %v, want 7.0", actualHours)
		}
		// 28800000 / 3600000 = 8.0
		if needHours != 8.0 {
			t.Errorf("needHours = %v, want 8.0", needHours)
		}
	})

	t.Run("workout_summary_computations", func(t *testing.T) {
		var day string
		var durationMins, distanceKm float64
		err := db.QueryRow(`SELECT day, duration_minutes, distance_km FROM workout_summary`).Scan(&day, &durationMins, &distanceKm)
		if err != nil {
			t.Fatalf("query workout_summary: %v", err)
		}
		if durationMins != 60.0 {
			t.Errorf("durationMins = %v, want 60.0", durationMins)
		}
		if distanceKm != 5.0 {
			t.Errorf("distanceKm = %v, want 5.0", distanceKm)
		}
	})

	t.Run("daily_strain_shape", func(t *testing.T) {
		rows, err := db.Query(`SELECT day, strain, kilojoule FROM daily_strain`)
		if err != nil {
			t.Fatalf("query daily_strain: %v", err)
		}
		defer rows.Close()
		cols, _ := rows.Columns()
		if cols[0] != "day" || cols[1] != "strain" {
			t.Errorf("unexpected columns: %v", cols)
		}
	})
}
