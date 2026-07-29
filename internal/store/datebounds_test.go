package store

import (
	"testing"
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/models"
)

func TestGetSleepTrend_IncludesEndDay(t *testing.T) {
	db := openTestDB(t)

	if err := db.SaveSleeps([]models.Sleep{
		{
			ID: "1", UserID: 1,
			CreatedAt: "2024-01-14T22:00:00Z", UpdatedAt: "2024-01-15T06:00:00Z",
			Start: "2024-01-14T22:00:00Z", End: "2024-01-15T06:00:00Z",
			Nap: false, ScoreState: "SCORED",
			Score: &models.SleepScore{
				StageSummary: models.SleepStageSummary{
					TotalInBedTimeMilli: 28800000,
					TotalAwakeTimeMilli: 0,
				},
			},
		},
		{
			ID: "2", UserID: 1,
			CreatedAt: "2024-01-15T22:00:00Z", UpdatedAt: "2024-01-16T06:00:00Z",
			Start: "2024-01-15T22:00:00Z", End: "2024-01-16T06:00:00Z",
			Nap: false, ScoreState: "SCORED",
			Score: &models.SleepScore{
				StageSummary: models.SleepStageSummary{
					TotalInBedTimeMilli: 25200000,
					TotalAwakeTimeMilli: 0,
				},
			},
		},
	}); err != nil {
		t.Fatalf("SaveSleeps: %v", err)
	}

	// Bare YYYY-MM-DD to must include the entire end calendar day.
	points, err := db.GetSleepTrend("2024-01-14", "2024-01-15")
	if err != nil {
		t.Fatalf("GetSleepTrend: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("len(points) = %d, want 2 (end day included)", len(points))
	}
	if points[0].Date != "2024-01-14" || points[1].Date != "2024-01-15" {
		t.Fatalf("dates = %s, %s; want 2024-01-14, 2024-01-15", points[0].Date, points[1].Date)
	}
}

func TestGetCorrelationData_ExcludesNaps(t *testing.T) {
	db := openTestDB(t)

	now := time.Now().UTC()
	day := now.Format("2006-01-02")
	start := now.Format(time.RFC3339)

	if err := db.SaveSleeps([]models.Sleep{
		{
			ID: "main", UserID: 1,
			CreatedAt: start, UpdatedAt: start,
			Start: start, End: start,
			Nap: false, ScoreState: "SCORED",
			Score: &models.SleepScore{
				StageSummary:       models.SleepStageSummary{TotalInBedTimeMilli: 28800000},
				SleepEfficiencyPct: 90,
			},
		},
		{
			ID: "nap", UserID: 1,
			CreatedAt: start, UpdatedAt: start,
			Start: now.Add(6 * time.Hour).Format(time.RFC3339), End: start,
			Nap: true, ScoreState: "SCORED",
			Score: &models.SleepScore{
				StageSummary:       models.SleepStageSummary{TotalInBedTimeMilli: 1800000},
				SleepEfficiencyPct: 50,
			},
		},
	}); err != nil {
		t.Fatalf("SaveSleeps: %v", err)
	}

	if err := db.SaveRecoveries([]models.Recovery{{
		CycleID: 1, UserID: 1,
		CreatedAt: start, UpdatedAt: start,
		ScoreState: "SCORED",
		Score:      &models.RecoveryScore{RecoveryScore: 70},
	}}); err != nil {
		t.Fatalf("SaveRecoveries: %v", err)
	}

	points, err := db.GetCorrelationData("recovery", "sleep_efficiency")
	if err != nil {
		t.Fatalf("GetCorrelationData: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("len(points) = %d, want 1 (nap excluded)", len(points))
	}
	if points[0].Y != 90 {
		t.Fatalf("sleep_efficiency = %v, want 90 (main sleep only)", points[0].Y)
	}

	// Same-table sleep metrics also exclude naps.
	same, err := db.GetCorrelationData("sleep_duration", "sleep_efficiency")
	if err != nil {
		t.Fatalf("GetCorrelationData same-table: %v", err)
	}
	if len(same) != 1 {
		t.Fatalf("same-table len = %d, want 1", len(same))
	}
	_ = day
}

func TestDateBounds_EndOfDay(t *testing.T) {
	from, to, err := DateBounds("2024-01-15", "2024-01-15")
	if err != nil {
		t.Fatalf("DateBounds: %v", err)
	}
	if from != "2024-01-15T00:00:00Z" {
		t.Fatalf("from = %q", from)
	}
	if to != "2024-01-15T23:59:59Z" {
		t.Fatalf("to = %q, want end of day", to)
	}
}
