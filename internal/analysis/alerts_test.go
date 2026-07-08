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

func openAlertsDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("Open DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestCheckAlertsLogic(t *testing.T) {
	db := openAlertsDB(t)

	now := time.Now().UTC().Format(time.RFC3339)
	if err := db.SaveRecoveries([]models.Recovery{
		{CycleID: 1, UserID: 123, CreatedAt: now, ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 10}},
	}); err != nil {
		t.Fatalf("SaveRecoveries: %v", err)
	}

	cfg := &config.Config{
		Alerts: config.Alerts{
			LowRecovery: 33,
			Enabled:     true,
		},
	}

	alerts, err := CheckAlerts(db, cfg)
	if err != nil {
		t.Fatalf("CheckAlerts: %v", err)
	}
	if len(alerts) == 0 {
		t.Error("expected alerts for low recovery")
	}

	cfg.Alerts.Enabled = false
	alerts, err = CheckAlerts(db, cfg)
	if err != nil {
		t.Fatalf("CheckAlerts disabled: %v", err)
	}
	if len(alerts) != 0 {
		t.Error("expected no alerts when disabled")
	}
}

func TestCheckAlerts_HighStrain(t *testing.T) {
	db := openAlertsDB(t)

	now := time.Now().UTC().Format(time.RFC3339)
	if err := db.SaveCycles([]models.Cycle{
		{ID: 1, UserID: 123, Start: now, ScoreState: "SCORED", Score: &models.CycleScore{Strain: 19.5}},
	}); err != nil {
		t.Fatalf("SaveCycles: %v", err)
	}

	cfg := &config.Config{
		Alerts: config.Alerts{
			HighStrain: 18,
			Enabled:    true,
		},
	}

	alerts, err := CheckAlerts(db, cfg)
	if err != nil {
		t.Fatalf("CheckAlerts: %v", err)
	}
	if len(alerts) == 0 {
		t.Error("expected alerts for high strain")
	}
}

func TestCheckAlerts_NoDataDoesNotAlert(t *testing.T) {
	db := openAlertsDB(t)

	cfg := &config.Config{
		Alerts: config.Alerts{
			LowRecovery: 33,
			HighStrain:  18,
			Enabled:     true,
		},
	}

	alerts, err := CheckAlerts(db, cfg)
	if err != nil {
		t.Fatalf("CheckAlerts: %v", err)
	}
	if len(alerts) != 0 {
		t.Fatalf("len(alerts) = %d, want 0: %+v", len(alerts), alerts)
	}
}

func TestCheckAlerts_MissingRecoveryDoesNotBlockHighStrain(t *testing.T) {
	db := openAlertsDB(t)

	now := time.Now().UTC().Format(time.RFC3339)
	if err := db.SaveCycles([]models.Cycle{
		{ID: 1, UserID: 123, Start: now, ScoreState: "SCORED", Score: &models.CycleScore{Strain: 19.5}},
	}); err != nil {
		t.Fatalf("SaveCycles: %v", err)
	}

	cfg := &config.Config{
		Alerts: config.Alerts{
			LowRecovery: 33,
			HighStrain:  18,
			Enabled:     true,
		},
	}

	alerts, err := CheckAlerts(db, cfg)
	if err != nil {
		t.Fatalf("CheckAlerts: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("len(alerts) = %d, want 1: %+v", len(alerts), alerts)
	}
	if alerts[0].Message != "High strain: 19.5 (threshold: 18)" {
		t.Fatalf("alert message = %q, want high strain alert", alerts[0].Message)
	}
}

func TestCheckAlerts_MissingStrainDoesNotAddAlert(t *testing.T) {
	db := openAlertsDB(t)

	now := time.Now().UTC().Format(time.RFC3339)
	if err := db.SaveRecoveries([]models.Recovery{
		{CycleID: 1, UserID: 123, CreatedAt: now, ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 20}},
	}); err != nil {
		t.Fatalf("SaveRecoveries: %v", err)
	}

	cfg := &config.Config{
		Alerts: config.Alerts{
			LowRecovery: 33,
			HighStrain:  18,
			Enabled:     true,
		},
	}

	alerts, err := CheckAlerts(db, cfg)
	if err != nil {
		t.Fatalf("CheckAlerts: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("len(alerts) = %d, want 1: %+v", len(alerts), alerts)
	}
	if alerts[0].Message != "Low recovery: 20% (threshold: 33%)" {
		t.Fatalf("alert message = %q, want low recovery alert", alerts[0].Message)
	}
}

func TestCheckAlerts_UsesLatestRecoveryToday(t *testing.T) {
	db := openAlertsDB(t)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	early := today.Add(6 * time.Hour).Format(time.RFC3339)
	late := today.Add(12 * time.Hour).Format(time.RFC3339)

	if err := db.SaveRecoveries([]models.Recovery{
		{CycleID: 1, UserID: 123, CreatedAt: early, ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 10}},
		{CycleID: 2, UserID: 123, CreatedAt: late, ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 80}},
	}); err != nil {
		t.Fatalf("SaveRecoveries: %v", err)
	}

	cfg := &config.Config{
		Alerts: config.Alerts{
			LowRecovery: 33,
			Enabled:     true,
		},
	}

	alerts, err := CheckAlerts(db, cfg)
	if err != nil {
		t.Fatalf("CheckAlerts: %v", err)
	}
	if len(alerts) != 0 {
		t.Fatalf("expected no alerts when latest recovery is healthy, got %+v", alerts)
	}
}

func TestCheckAlerts_DateOnlyToIncludesTodayTimestamps(t *testing.T) {
	db := openAlertsDB(t)

	// Mid-day timestamp must still match today's date-only upper bound.
	now := time.Now().UTC()
	midDay := time.Date(now.Year(), now.Month(), now.Day(), 15, 30, 0, 0, time.UTC).Format(time.RFC3339)
	if err := db.SaveRecoveries([]models.Recovery{
		{CycleID: 1, UserID: 123, CreatedAt: midDay, ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 15}},
	}); err != nil {
		t.Fatalf("SaveRecoveries: %v", err)
	}

	cfg := &config.Config{
		Alerts: config.Alerts{LowRecovery: 33, Enabled: true},
	}
	alerts, err := CheckAlerts(db, cfg)
	if err != nil {
		t.Fatalf("CheckAlerts: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("len(alerts) = %d, want 1 (date bound must include mid-day)", len(alerts))
	}
}
