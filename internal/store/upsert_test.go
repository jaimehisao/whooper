package store

import (
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/models"
)

func TestSaveCycles_PendingScoreDoesNotWipeScoredMetrics(t *testing.T) {
	db := openTestDB(t)

	scored := models.Cycle{
		ID: 1, UserID: 1, CreatedAt: "2024-01-15T00:00:00Z", UpdatedAt: "2024-01-15T00:00:00Z",
		Start: "2024-01-15T00:00:00Z", End: "2024-01-16T00:00:00Z", Days: 1, ScoreState: "SCORED",
		Score: &models.CycleScore{Strain: 12.5, Kilojoule: 2000, AverageHeartRate: 70, MaxHeartRate: 150},
	}
	if err := db.SaveCycles([]models.Cycle{scored}); err != nil {
		t.Fatalf("SaveCycles scored: %v", err)
	}

	pending := scored
	pending.ScoreState = "PENDING_SCORE"
	pending.Score = nil
	pending.UpdatedAt = "2024-01-15T12:00:00Z"
	if err := db.SaveCycles([]models.Cycle{pending}); err != nil {
		t.Fatalf("SaveCycles pending: %v", err)
	}

	got, err := db.ListCycles("", "")
	if err != nil || len(got) != 1 {
		t.Fatalf("ListCycles: len=%d err=%v", len(got), err)
	}
	if got[0].ScoreState != "PENDING_SCORE" {
		t.Fatalf("score_state = %q, want PENDING_SCORE", got[0].ScoreState)
	}
	if got[0].Score != nil {
		t.Fatalf("Score should be nil for PENDING_SCORE, got %+v", got[0].Score)
	}

	var strain float64
	if err := db.QueryRow(`SELECT strain FROM cycle WHERE id = 1`).Scan(&strain); err != nil {
		t.Fatalf("scan strain: %v", err)
	}
	if strain != 12.5 {
		t.Fatalf("strain = %v, want 12.5 preserved", strain)
	}
}

func TestSaveRecoveries_OmittedSpO2DoesNotClobber(t *testing.T) {
	db := openTestDB(t)

	withSpO2 := models.Recovery{
		CycleID: 1, SleepID: "s1", UserID: 1,
		CreatedAt: "2024-01-15T06:00:00Z", UpdatedAt: "2024-01-15T06:00:00Z", ScoreState: "SCORED",
		Score: &models.RecoveryScore{
			RecoveryScore: 80, RestingHeartRate: 50, HRVRmssd: 60,
			SpO2Percentage: models.FloatPtr(98), SkinTempCelsius: models.FloatPtr(33.1),
		},
	}
	if err := db.SaveRecoveries([]models.Recovery{withSpO2}); err != nil {
		t.Fatalf("SaveRecoveries with spo2: %v", err)
	}

	withoutOptional := withSpO2
	withoutOptional.UpdatedAt = "2024-01-15T07:00:00Z"
	withoutOptional.Score = &models.RecoveryScore{
		RecoveryScore: 82, RestingHeartRate: 51, HRVRmssd: 62,
		// SpO2 and skin temp omitted
	}
	if err := db.SaveRecoveries([]models.Recovery{withoutOptional}); err != nil {
		t.Fatalf("SaveRecoveries without optional: %v", err)
	}

	got, err := db.ListRecoveries("", "")
	if err != nil || len(got) != 1 || got[0].Score == nil {
		t.Fatalf("ListRecoveries: %+v err=%v", got, err)
	}
	if got[0].Score.RecoveryScore != 82 {
		t.Fatalf("recovery_score = %v, want 82", got[0].Score.RecoveryScore)
	}
	if got[0].Score.SpO2OrZero() != 98 {
		t.Fatalf("spo2 = %v, want 98 preserved", got[0].Score.SpO2OrZero())
	}
	if got[0].Score.SkinTempOrZero() != 33.1 {
		t.Fatalf("skin_temp = %v, want 33.1 preserved", got[0].Score.SkinTempOrZero())
	}
}
