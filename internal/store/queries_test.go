package store

import (
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/models"
)

func TestMetricColumn(t *testing.T) {
	tests := []struct {
		metric  string
		wantCol string
		wantTab string
		wantErr bool
	}{
		{"recovery", "recovery_score", "recovery", false},
		{"hrv", "hrv_rmssd", "recovery", false},
		{"rhr", "resting_heart_rate", "recovery", false},
		{"strain", "strain", "cycle", false},
		{"sleep_duration", "(total_in_bed_time_milli - total_awake_time_milli - COALESCE(total_no_data_time_milli, 0))", "sleep", false},
		{"sleep_efficiency", "sleep_efficiency_pct", "sleep", false},
		{"invalid_metric", "", "", true},
	}

	for _, tt := range tests {
		col, tab, err := metricColumn(tt.metric)
		if tt.wantErr {
			if err == nil {
				t.Errorf("metricColumn(%q) expected error, got nil", tt.metric)
			}
			continue
		}
		if err != nil {
			t.Errorf("metricColumn(%q) unexpected error: %v", tt.metric, err)
			continue
		}
		if col != tt.wantCol || tab != tt.wantTab {
			t.Errorf("metricColumn(%q) = (%q, %q), want (%q, %q)",
				tt.metric, col, tab, tt.wantCol, tt.wantTab)
		}
	}
}

func TestDateColumn(t *testing.T) {
	tests := []struct {
		table string
		want  string
	}{
		{"recovery", "created_at"},
		{"cycle", "start"},
		{"sleep", "start"},
		{"unknown", "start"},
	}

	for _, tt := range tests {
		got := dateColumn(tt.table)
		if got != tt.want {
			t.Errorf("dateColumn(%q) = %q, want %q", tt.table, got, tt.want)
		}
	}
}

func TestGetSleepTrend(t *testing.T) {
	db := openTestDB(t)

	err := db.SaveSleeps([]models.Sleep{
		{
			ID: "1", UserID: 1,
			CreatedAt:  "2024-01-15T00:00:00Z",
			UpdatedAt:  "2024-01-15T07:00:00Z",
			Start:      "2024-01-15T00:00:00Z",
			End:        "2024-01-15T07:00:00Z",
			Nap:        false,
			ScoreState: "SCORED",
			Score: &models.SleepScore{
				StageSummary: models.SleepStageSummary{
					TotalInBedTimeMilli: 28800000,
					TotalAwakeTimeMilli: 3600000,
				},
				SleepEfficiencyPct:  90,
				SleepPerformancePct: 88,
				SleepConsistencyPct: 85,
			},
		},
		{
			ID: "2", UserID: 1,
			CreatedAt:  "2024-01-15T12:00:00Z",
			UpdatedAt:  "2024-01-15T12:30:00Z",
			Start:      "2024-01-15T12:00:00Z",
			End:        "2024-01-15T12:30:00Z",
			Nap:        true,
			ScoreState: "SCORED",
			Score: &models.SleepScore{
				StageSummary: models.SleepStageSummary{
					TotalInBedTimeMilli: 1800000,
					TotalAwakeTimeMilli: 300000,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("SaveSleeps error = %v", err)
	}

	points, err := db.GetSleepTrend("2024-01-01", "2024-12-31")
	if err != nil {
		t.Fatalf("GetSleepTrend error = %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("len(points) = %d, want 1", len(points))
	}
	if points[0].DurationMilli != 25200000 {
		t.Fatalf("DurationMilli = %d, want 25200000", points[0].DurationMilli)
	}
}

func TestGetCorrelationDataAcrossTablesAndErrors(t *testing.T) {
	db := openTestDB(t)

	if err := db.SaveRecoveries([]models.Recovery{{
		CycleID: 1, UserID: 1,
		CreatedAt:  "2024-01-15T06:00:00Z",
		UpdatedAt:  "2024-01-15T06:00:00Z",
		ScoreState: "SCORED",
		Score:      &models.RecoveryScore{RecoveryScore: 75},
	}}); err != nil {
		t.Fatalf("SaveRecoveries error = %v", err)
	}

	if err := db.SaveSleeps([]models.Sleep{{
		ID: "1", UserID: 1,
		CreatedAt:  "2024-01-15T00:00:00Z",
		UpdatedAt:  "2024-01-15T06:00:00Z",
		Start:      "2024-01-15T00:00:00Z",
		End:        "2024-01-15T06:00:00Z",
		Nap:        false,
		ScoreState: "SCORED",
		Score: &models.SleepScore{
			StageSummary: models.SleepStageSummary{
				TotalInBedTimeMilli: 28800000,
				TotalAwakeTimeMilli: 3600000,
			},
		},
	}}); err != nil {
		t.Fatalf("SaveSleeps error = %v", err)
	}

	points, err := db.GetCorrelationData("recovery", "sleep_efficiency")
	if err != nil {
		t.Fatalf("GetCorrelationData cross table error = %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("len(points) = %d, want 1", len(points))
	}

	if _, err := db.GetCorrelationData("invalid", "sleep_efficiency"); err == nil {
		t.Fatal("expected error for invalid X metric")
	}
	if _, err := db.GetCorrelationData("recovery", "invalid"); err == nil {
		t.Fatal("expected error for invalid Y metric")
	}
}

func TestGetCorrelationDataAcrossTablesDailyAggregation(t *testing.T) {
	db := openTestDB(t)

	if err := db.SaveRecoveries([]models.Recovery{
		{
			CycleID:    1,
			UserID:     1,
			CreatedAt:  "2024-01-15T01:00:00Z",
			UpdatedAt:  "2024-01-15T01:00:00Z",
			ScoreState: "SCORED",
			Score:      &models.RecoveryScore{RecoveryScore: 60},
		},
		{
			CycleID:    2,
			UserID:     1,
			CreatedAt:  "2024-01-15T12:00:00Z",
			UpdatedAt:  "2024-01-15T12:00:00Z",
			ScoreState: "SCORED",
			Score:      &models.RecoveryScore{RecoveryScore: 80},
		},
	}); err != nil {
		t.Fatalf("SaveRecoveries error = %v", err)
	}

	if err := db.SaveCycles([]models.Cycle{
		{
			ID:         1,
			UserID:     1,
			CreatedAt:  "2024-01-15T06:00:00Z",
			UpdatedAt:  "2024-01-15T06:00:00Z",
			Start:      "2024-01-15T06:00:00Z",
			End:        "2024-01-15T07:00:00Z",
			ScoreState: "SCORED",
			Score:      &models.CycleScore{Strain: 10},
		},
		{
			ID:         2,
			UserID:     1,
			CreatedAt:  "2024-01-15T18:00:00Z",
			UpdatedAt:  "2024-01-15T18:00:00Z",
			Start:      "2024-01-15T18:00:00Z",
			End:        "2024-01-15T19:00:00Z",
			ScoreState: "SCORED",
			Score:      &models.CycleScore{Strain: 14},
		},
	}); err != nil {
		t.Fatalf("SaveCycles error = %v", err)
	}

	points, err := db.GetCorrelationData("recovery", "strain")
	if err != nil {
		t.Fatalf("GetCorrelationData error = %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("len(points) = %d, want 1", len(points))
	}
	if points[0].X != 70 {
		t.Fatalf("aggregated recovery = %v, want 70", points[0].X)
	}
	if points[0].Y != 12 {
		t.Fatalf("aggregated strain = %v, want 12", points[0].Y)
	}
}
