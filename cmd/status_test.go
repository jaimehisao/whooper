package cmd

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/oauth2"

	"git.infra.hisao.org/hisao/whooper/internal/auth"
	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/models"
	"git.infra.hisao.org/hisao/whooper/internal/store"
)

func setupStatusEnv(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), filepath.Join(tmpDir, "whooper.db"))
	return tmpDir
}

func TestBuildStatusReport(t *testing.T) {
	_ = setupStatusEnv(t)
	if err := config.Save(&config.Config{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost:8484/callback"}); err != nil {
		t.Fatalf("Save config: %v", err)
	}
	if err := auth.SaveToken(config.TokenPath(), &oauth2.Token{AccessToken: "token"}); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatalf("Open DB: %v", err)
	}
	if err := db.SaveCycles([]models.Cycle{{ID: 1, UserID: 1, Start: "2024-01-01T00:00:00Z"}}); err != nil {
		t.Fatalf("SaveCycles: %v", err)
	}
	if err := db.SetSyncState("cycles", "2024-01-02T00:00:00Z"); err != nil {
		t.Fatalf("SetSyncState: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close DB: %v", err)
	}

	report := buildStatusReport()
	if !report.ClientIDConfigured || !report.ClientSecretConfigured {
		t.Fatalf("expected configured credentials: %+v", report)
	}
	if !report.TokenPresent || !report.DBOpen {
		t.Fatalf("expected token and DB: %+v", report)
	}
	if report.RecordCounts["cycles"] != 1 {
		t.Fatalf("cycles count = %d, want 1", report.RecordCounts["cycles"])
	}
	if report.LastSync["cycles"] != "2024-01-02T00:00:00Z" {
		t.Fatalf("cycles last sync = %q", report.LastSync["cycles"])
	}
}

func TestStatusTextAndJSONOutput(t *testing.T) {
	_ = setupStatusEnv(t)
	if err := config.Save(&config.Config{ClientID: "id", RedirectURL: "http://localhost:8484/callback"}); err != nil {
		t.Fatalf("Save config: %v", err)
	}

	report := buildStatusReport()

	var text bytes.Buffer
	writeStatusText(&text, report)
	if !strings.Contains(text.String(), "Client ID configured: true") {
		t.Fatalf("status text missing configured client id:\n%s", text.String())
	}

	var jsonOut bytes.Buffer
	if err := writeStatusJSON(&jsonOut, report); err != nil {
		t.Fatalf("writeStatusJSON: %v", err)
	}
	var decoded statusReport
	if err := json.Unmarshal(jsonOut.Bytes(), &decoded); err != nil {
		t.Fatalf("status JSON decode: %v\n%s", err, jsonOut.String())
	}
	if !decoded.ClientIDConfigured {
		t.Fatalf("decoded status missing client id flag: %+v", decoded)
	}
}

func TestCountStatusRowsRejectsUnknownEntity(t *testing.T) {
	_ = setupStatusEnv(t)
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatalf("Open DB: %v", err)
	}
	defer db.Close()

	if _, err := countStatusRows(db, "bad"); err == nil {
		t.Fatal("expected error for unknown status entity")
	}
}
