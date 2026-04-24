package analysis

import (
	"testing"
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/models"
	"git.infra.hisao.org/hisao/whooper/internal/store"
)

func TestEvaluateAlerts_LowRecovery(t *testing.T) {
	alerts := EvaluateAlerts(10, 10, 33, 18)
	found := false
	for _, a := range alerts {
		if a.Level == "critical" {
			found = true
		}
	}
	if !found {
		t.Error("expected critical alert for recovery below half threshold")
	}
}

func TestEvaluateAlerts_WarningRecovery(t *testing.T) {
	alerts := EvaluateAlerts(25, 10, 33, 18)
	found := false
	for _, a := range alerts {
		if a.Level == "warning" {
			found = true
		}
	}
	if !found {
		t.Error("expected warning alert for low recovery")
	}
}

func TestEvaluateAlerts_HighStrain(t *testing.T) {
	alerts := EvaluateAlerts(80, 20, 33, 18)
	found := false
	for _, a := range alerts {
		if a.Level == "warning" {
			found = true
		}
	}
	if !found {
		t.Error("expected warning alert for high strain")
	}
}

func TestEvaluateAlerts_NoAlerts(t *testing.T) {
	alerts := EvaluateAlerts(80, 10, 33, 18)
	if len(alerts) != 0 {
		t.Errorf("expected no alerts, got %d", len(alerts))
	}
}

func TestEvaluateAlerts_BothAlerts(t *testing.T) {
	alerts := EvaluateAlerts(20, 20, 33, 18)
	if len(alerts) != 2 {
		t.Errorf("expected 2 alerts, got %d", len(alerts))
	}
}

func TestCheckAlertsLogic(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, _ := store.Open(dbPath)
	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	db.SaveRecoveries([]models.Recovery{
		{CycleID: 1, UserID: 123, CreatedAt: now, ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 10}},
	})

	cfg := &config.Config{
		Alerts: config.Alerts{
			LowRecovery: 33,
			Enabled:     true,
		},
	}

	alerts := CheckAlerts(db, cfg)
	if len(alerts) == 0 {
		t.Error("expected alerts for low recovery")
	}

	// Test disabled
	cfg.Alerts.Enabled = false
	alerts = CheckAlerts(db, cfg)
	if len(alerts) != 0 {
		t.Error("expected no alerts when disabled")
	}
}

func TestCheckAlerts_HighStrain(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, _ := store.Open(dbPath)
	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	db.SaveCycles([]models.Cycle{
		{ID: 1, UserID: 123, Start: now, ScoreState: "SCORED", Score: &models.CycleScore{Strain: 19.5}},
	})

	cfg := &config.Config{
		Alerts: config.Alerts{
			HighStrain: 18,
			Enabled:    true,
		},
	}

	alerts := CheckAlerts(db, cfg)
	if len(alerts) == 0 {
		t.Error("expected alerts for high strain")
	}
}
