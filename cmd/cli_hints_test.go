package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

func TestSyncCommandHints(t *testing.T) {
	// Save original functions
	origLoadConfig := syncLoadConfig
	origLoadToken := syncLoadToken
	origOpenDB := syncOpenDB
	defer func() {
		syncLoadConfig = origLoadConfig
		syncLoadToken = origLoadToken
		syncOpenDB = origOpenDB
	}()

	t.Run("missing config hint", func(t *testing.T) {
		syncLoadConfig = func() (*config.Config, error) {
			return &config.Config{}, nil
		}
		cmd := &cobra.Command{}
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		err := runSyncOnce(cmd)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "whooper config set client-id") {
			t.Errorf("expected config hint, got: %v", err)
		}
	})

	t.Run("failed db open hint", func(t *testing.T) {
		syncLoadConfig = func() (*config.Config, error) {
			return &config.Config{ClientID: "id", ClientSecret: "secret"}, nil
		}
		syncLoadToken = func(string) (*oauth2.Token, error) {
			return &oauth2.Token{}, nil
		}
		syncOpenDB = func(string) (*store.DB, error) {
			return nil, errors.New("cannot open")
		}
		cmd := &cobra.Command{}
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		err := runSyncOnce(cmd)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Hint: run 'whooper login' or 'whooper sync' to initialize the database") {
			t.Errorf("expected db hint, got: %v", err)
		}
	})
}

func TestStatusCommandHints(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), filepath.Join(tmpDir, "whooper.db"))

	t.Run("missing config hint in text output", func(t *testing.T) {
		report := statusReport{
			ClientIDConfigured:     false,
			ClientSecretConfigured: false,
		}
		var out bytes.Buffer
		writeStatusText(&out, report)
		if !strings.Contains(out.String(), "whooper config set client-id") {
			t.Errorf("expected config hint, got:\n%s", out.String())
		}
	})

	t.Run("missing token hint in text output", func(t *testing.T) {
		report := statusReport{
			ClientIDConfigured:     true,
			ClientSecretConfigured: true,
			TokenPresent:           false,
		}
		var out bytes.Buffer
		writeStatusText(&out, report)
		if !strings.Contains(out.String(), "whooper login") {
			t.Errorf("expected login hint, got:\n%s", out.String())
		}
	})

	t.Run("empty database hint in text output", func(t *testing.T) {
		report := statusReport{
			ClientIDConfigured:     true,
			ClientSecretConfigured: true,
			TokenPresent:           true,
			DBOpen:                 true,
			RecordCounts:           map[string]int{"cycles": 0, "recoveries": 0, "sleeps": 0, "workouts": 0},
		}
		var out bytes.Buffer
		writeStatusText(&out, report)
		if !strings.Contains(out.String(), "Local database is empty") {
			t.Errorf("expected empty db hint, got:\n%s", out.String())
		}
	})
}

func TestDoctorCommandHints(t *testing.T) {
	deps := defaultDoctorDeps()
	deps.loadConfig = func() (*config.Config, error) {
		return &config.Config{ClientID: ""}, nil
	}

	report := buildDoctorReport(deps, false)
	foundHint := false
	for _, check := range report.Checks {
		if check.Name == "client-id configured" && strings.Contains(check.Error, "run 'whooper config set client-id") {
			foundHint = true
			break
		}
	}
	if !foundHint {
		t.Errorf("expected client-id hint in doctor report, checks: %+v", report.Checks)
	}
}

func TestExportCommandHints(t *testing.T) {
	resetExportFlags()
	defer resetExportFlags()
	tmpDir := t.TempDir()
	// Use a path that will definitely fail to open as a DB (by putting a file where a directory should be)
	blockedDir := filepath.Join(tmpDir, "blocked")
	if err := os.WriteFile(blockedDir, []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(blockedDir, "whooper.db")
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), dbPath)

	t.Run("failed db open hint", func(t *testing.T) {
		err := exportCmd.RunE(exportCmd, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Hint: run 'whooper sync' or 'whooper login' first to initialize the database") {
			t.Errorf("expected db hint in export, got: %v", err)
		}
	})
}
