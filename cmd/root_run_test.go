package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/models"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	"golang.org/x/oauth2"
)

func TestConfigRunE(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), filepath.Join(tmpDir, "whooper.db"))
	
	// Mock config
	config.Save(&config.Config{ClientID: "test"})

	var buf bytes.Buffer
	configCmd.SetOut(&buf)
	defer configCmd.SetOut(nil)

	err := configCmd.RunE(configCmd, []string{})
	if err != nil {
		t.Fatalf("configCmd.RunE error = %v", err)
	}

	if !strings.Contains(buf.String(), "client_id:     test") {
		t.Errorf("unexpected output: %s", buf.String())
	}
}

func TestConfigSetRunE(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), filepath.Join(tmpDir, "whooper.db"))

	err := configSetCmd.RunE(configSetCmd, []string{"client-id", "new-id"})
	if err != nil {
		t.Fatalf("configSetCmd.RunE error = %v", err)
	}

	cfg, _ := config.Load()
	if cfg.ClientID != "new-id" {
		t.Errorf("got %s, want new-id", cfg.ClientID)
	}
}

func TestVersionRun(t *testing.T) {
	var buf bytes.Buffer
	versionCmd.SetOut(&buf)
	defer versionCmd.SetOut(nil)

	SetVersion("1.2.3")
	versionCmd.Run(versionCmd, []string{})
	
	if !strings.Contains(buf.String(), "1.2.3") {
		t.Errorf("unexpected output: %s", buf.String())
	}
}

func TestExportRunE(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "whooper.db")
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), dbPath)

	// Seed data
	db, _ := store.Open(dbPath)
	db.SaveRecoveries([]models.Recovery{
		{CycleID: 1, UserID: 123, CreatedAt: "2024-01-15T00:00:00Z", ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 75}},
	})
	db.Close()

	origEntity := exportEntity
	origFormat := exportFormat
	defer func() { exportEntity = origEntity; exportFormat = origFormat }()

	exportEntity = "recoveries"
	exportFormat = "json"

	var buf bytes.Buffer
	exportCmd.SetOut(&buf)
	defer exportCmd.SetOut(nil)

	err := exportCmd.RunE(exportCmd, []string{})
	if err != nil {
		t.Fatalf("exportCmd.RunE error = %v", err)
	}

	exportFormat = "csv"
	buf.Reset()
	err = exportCmd.RunE(exportCmd, []string{})
	if err != nil {
		t.Fatalf("exportCmd.RunE error = %v", err)
	}

	// Invalid entity
	exportEntity = "invalid"
	err = exportCmd.RunE(exportCmd, []string{})
	if err == nil {
		t.Error("expected error for invalid entity")
	}
}

func TestConfigSetRunE_InvalidKey(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), filepath.Join(tmpDir, "whooper.db"))

	err := configSetCmd.RunE(configSetCmd, []string{"invalid", "val"})
	if err == nil {
		t.Error("expected error for invalid key")
	}
}

func TestSyncRunE_NoToken(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), filepath.Join(tmpDir, "whooper.db"))
	config.Save(&config.Config{ClientID: "id", ClientSecret: "secret"})

	err := syncCmd.RunE(syncCmd, []string{})
	if err == nil {
		t.Error("expected error due to missing token")
	}
}

func TestLoginRunE_Success(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), filepath.Join(tmpDir, "whooper.db"))
	
	config.Save(&config.Config{ClientID: "id", ClientSecret: "secret"})

	origFunc := oauthFlowFunc
	oauthFlowFunc = func(*oauth2.Config) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: "fake"}, nil
	}
	defer func() { oauthFlowFunc = origFunc }()

	err := loginCmd.RunE(loginCmd, []string{})
	if err != nil {
		t.Fatalf("loginCmd.RunE error = %v", err)
	}
}

func TestDoctorRunE(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "whooper.db")
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), dbPath)

	var buf bytes.Buffer
	doctorCmd.SetOut(&buf)
	defer doctorCmd.SetOut(nil)

	err := doctorCmd.RunE(doctorCmd, []string{})
	
	// Expected to fail doctor checks due to missing client-id etc.
	if err == nil {
		t.Error("expected doctor command to fail on missing config")
	}
}
