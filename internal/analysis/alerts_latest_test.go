package analysis

import (
	"testing"
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/models"
	"git.infra.hisao.org/hisao/whooper/internal/store"
)

func TestCheckAlerts_UsesLatestSameDaySample(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := store.Open(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	today := time.Now().UTC().Format("2006-01-02")
	early := today + "T08:00:00Z"
	late := today + "T18:00:00Z"

	// Early low recovery would alert; latest above threshold should not.
	// Daily average of 10 and 50 is 30 (would still warn) — proves we use latest.
	if err := db.SaveRecoveries([]models.Recovery{
		{
			CycleID: 1, UserID: 1,
			CreatedAt: early, UpdatedAt: early,
			ScoreState: "SCORED",
			Score:      &models.RecoveryScore{RecoveryScore: 10},
		},
		{
			CycleID: 2, UserID: 1,
			CreatedAt: late, UpdatedAt: late,
			ScoreState: "SCORED",
			Score:      &models.RecoveryScore{RecoveryScore: 50},
		},
	}); err != nil {
		t.Fatalf("SaveRecoveries: %v", err)
	}

	cfg := &config.Config{
		Alerts: config.Alerts{Enabled: true, LowRecovery: 33, HighStrain: 18},
	}
	alerts := CheckAlerts(db, cfg)
	if len(alerts) != 0 {
		t.Fatalf("len(alerts) = %d, want 0 (latest recovery 50 >= 33): %+v", len(alerts), alerts)
	}
}
