package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/models"
	"git.infra.hisao.org/hisao/whooper/internal/store"
)

type CLIEnv struct {
	tmpDir    string
	configDir string
	dbPath    string
}

func setupCLIEnv(t *testing.T) *CLIEnv {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".whooper")
	os.MkdirAll(configDir, 0o700)

	env := &CLIEnv{
		tmpDir:    tmpDir,
		configDir: configDir,
		dbPath:    filepath.Join(configDir, "whooper.db"),
	}

	config.SetTestPaths(configDir, filepath.Join(configDir, "config.yaml"), env.dbPath)

	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})

	return env
}

func setupTestDB(t *testing.T, env *CLIEnv) *store.DB {
	db, err := store.Open(env.dbPath)
	if err != nil {
		t.Fatalf("Open test DB: %v", err)
	}

	recoveries := []models.Recovery{
		{
			CycleID:    1,
			UserID:     123,
			CreatedAt:  "2024-01-15T00:00:00Z",
			ScoreState: "SCORED",
			Score: &models.RecoveryScore{
				RecoveryScore:    75,
				HRVRmssd:         45.0,
				RestingHeartRate: 55,
			},
		},
	}
	if err := db.SaveRecoveries(recoveries); err != nil {
		t.Fatalf("SaveRecoveries: %v", err)
	}

	cycles := []models.Cycle{
		{
			ID:         1,
			UserID:     123,
			Start:      "2024-01-15T00:00:00Z",
			End:        "2024-01-15T06:00:00Z",
			ScoreState: "SCORED",
			Score: &models.CycleScore{
				Strain: 12.5,
			},
		},
	}
	if err := db.SaveCycles(cycles); err != nil {
		t.Fatalf("SaveCycles: %v", err)
	}

	return db
}

func runCmd(t *testing.T, args []string) (string, error) {
	resetExportFlags()
	buf := bytes.Buffer{}
	errBuf := bytes.Buffer{}
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()

	combined := buf.String() + errBuf.String()
	return combined, err
}

func TestConfigCmd_Show(t *testing.T) {
	_ = setupCLIEnv(t)

	cfg := &config.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret-1234",
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save config: %v", err)
	}

	_, err := runCmd(t, []string{"config"})
	if err != nil {
		t.Fatalf("config command failed: %v", err)
	}
}

func TestConfigCmd_SetClientID(t *testing.T) {
	_ = setupCLIEnv(t)

	cfg := &config.Config{}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save config: %v", err)
	}

	_, err := runCmd(t, []string{"config", "set", "client-id", "new-client-id"})
	if err != nil {
		t.Fatalf("config set failed: %v", err)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	if loaded.ClientID != "new-client-id" {
		t.Errorf("ClientID = %s, want new-client-id", loaded.ClientID)
	}
}

func TestConfigCmd_SetClientSecret(t *testing.T) {
	_ = setupCLIEnv(t)

	cfg := &config.Config{}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save config: %v", err)
	}

	_, err := runCmd(t, []string{"config", "set", "client-secret", "new-secret"})
	if err != nil {
		t.Fatalf("config set failed: %v", err)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	if loaded.ClientSecret != "new-secret" {
		t.Errorf("ClientSecret = %s, want new-secret", loaded.ClientSecret)
	}
}

func TestConfigCmd_SetRedirectURL(t *testing.T) {
	_ = setupCLIEnv(t)

	cfg := &config.Config{}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save config: %v", err)
	}

	_, err := runCmd(t, []string{"config", "set", "redirect-url", "http://localhost:8484/callback"})
	if err != nil {
		t.Fatalf("config set failed: %v", err)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	if loaded.RedirectURL != "http://localhost:8484/callback" {
		t.Errorf("RedirectURL = %s, want http://localhost:8484/callback", loaded.RedirectURL)
	}
}

func TestConfigCmd_InvalidKey(t *testing.T) {
	_ = setupCLIEnv(t)

	_, err := runCmd(t, []string{"config", "set", "invalid-key", "value"})
	if err == nil {
		t.Error("expected error for invalid key")
	}
}

func TestConfigCmd_MissingArgs(t *testing.T) {
	_ = setupCLIEnv(t)

	_, err := runCmd(t, []string{"config", "set", "client-id"})
	if err == nil {
		t.Error("expected error for missing args")
	}
}

func TestExportCmd_InvalidEntity(t *testing.T) {
	env := setupCLIEnv(t)
	db := setupTestDB(t, env)
	defer db.Close()

	_, err := runCmd(t, []string{"export", "-e", "invalid-entity"})
	if err == nil {
		t.Error("expected error for invalid entity")
	}
}

func TestExportCmd_InvalidFormat(t *testing.T) {
	env := setupCLIEnv(t)
	db := setupTestDB(t, env)
	defer db.Close()

	_, err := runCmd(t, []string{"export", "-e", "recoveries", "-f", "invalid-format"})
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestExportCmd_WithData(t *testing.T) {
	env := setupCLIEnv(t)
	db := setupTestDB(t, env)
	defer db.Close()

	_, err := runCmd(t, []string{"export", "-e", "cycles"})
	if err != nil {
		t.Fatalf("export command failed: %v", err)
	}
}

func TestExportCmd_JSONFormat(t *testing.T) {
	env := setupCLIEnv(t)
	db := setupTestDB(t, env)
	defer db.Close()

	_, err := runCmd(t, []string{"export", "-e", "recoveries", "-f", "json"})
	if err != nil {
		t.Fatalf("export command failed: %v", err)
	}
}

func TestExportCmd_CSVFormat(t *testing.T) {
	env := setupCLIEnv(t)
	db := setupTestDB(t, env)
	defer db.Close()

	_, err := runCmd(t, []string{"export", "-e", "recoveries", "-f", "csv"})
	if err != nil {
		t.Fatalf("export command failed: %v", err)
	}
}

func TestExportCmd_EmptyDB(t *testing.T) {
	env := setupCLIEnv(t)

	db, err := store.Open(env.dbPath)
	if err != nil {
		t.Fatalf("Open test DB: %v", err)
	}
	defer db.Close()

	_, err = runCmd(t, []string{"export", "-e", "workouts", "-f", "json"})
	if err != nil {
		t.Fatalf("export command failed: %v", err)
	}
}
