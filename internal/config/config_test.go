package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}
	want := filepath.Join(home, ".whooper")
	if got := Dir(); got != want {
		t.Errorf("Dir() = %s, want %s", got, want)
	}
}

func TestPath(t *testing.T) {
	want := filepath.Join(Dir(), "config.yaml")
	if got := Path(); got != want {
		t.Errorf("Path() = %s, want %s", got, want)
	}
}

func TestDBPath(t *testing.T) {
	want := filepath.Join(Dir(), "whooper.db")
	if got := DBPath(); got != want {
		t.Errorf("DBPath() = %s, want %s", got, want)
	}
}

func TestTokenPath(t *testing.T) {
	want := filepath.Join(Dir(), "token.json")
	if got := TokenPath(); got != want {
		t.Errorf("TokenPath() = %s, want %s", got, want)
	}
}

func TestLoadNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "nonexistent.yaml")

	origPath := pathFunc
	pathFunc = func() string { return cfgPath }
	defer func() { pathFunc = origPath }()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.RedirectURL != defaultRedirectURL {
		t.Errorf("RedirectURL = %s, want %s", cfg.RedirectURL, defaultRedirectURL)
	}

	if cfg.Alerts.LowRecovery != defaultLowRecovery {
		t.Errorf("LowRecovery = %v, want %v", cfg.Alerts.LowRecovery, defaultLowRecovery)
	}

	if cfg.Alerts.HighStrain != defaultHighStrain {
		t.Errorf("HighStrain = %v, want %v", cfg.Alerts.HighStrain, defaultHighStrain)
	}

	if !cfg.Alerts.Enabled {
		t.Error("Enabled should be true by default")
	}
}

func TestLoadExisting(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `client_id: test-id
client_secret: test-secret
redirect_url: http://custom.callback/test
alerts:
  low_recovery: 25
  high_strain: 20
  enabled: false
`
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	origPath := pathFunc
	pathFunc = func() string { return cfgPath }
	defer func() { pathFunc = origPath }()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ClientID != "test-id" {
		t.Errorf("ClientID = %s, want test-id", cfg.ClientID)
	}
	if cfg.ClientSecret != "test-secret" {
		t.Errorf("ClientSecret = %s, want test-secret", cfg.ClientSecret)
	}
	if cfg.RedirectURL != "http://custom.callback/test" {
		t.Errorf("RedirectURL = %s, want http://custom.callback/test", cfg.RedirectURL)
	}
	if cfg.Alerts.LowRecovery != 25 {
		t.Errorf("LowRecovery = %v, want 25", cfg.Alerts.LowRecovery)
	}
	if cfg.Alerts.HighStrain != 20 {
		t.Errorf("HighStrain = %v, want 20", cfg.Alerts.HighStrain)
	}
	if cfg.Alerts.Enabled != false {
		t.Errorf("Enabled = %v, want false", cfg.Alerts.Enabled)
	}
}

func TestLoadEmptyRedirectURL(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := "client_id: test-id\n"
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	origPath := pathFunc
	pathFunc = func() string { return cfgPath }
	defer func() { pathFunc = origPath }()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.RedirectURL != defaultRedirectURL {
		t.Errorf("RedirectURL = %s, want default %s", cfg.RedirectURL, defaultRedirectURL)
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()

	origDir := dirFunc
	origPath := pathFunc

	dirFunc = func() string { return tmpDir }
	pathFunc = func() string { return filepath.Join(tmpDir, "config.yaml") }

	defer func() { dirFunc = origDir; pathFunc = origPath }()

	cfg := &Config{
		ClientID:     "saved-id",
		ClientSecret: "saved-secret",
		RedirectURL:  "http://saved.callback",
		Alerts: Alerts{
			LowRecovery: 40,
			HighStrain:  15,
			Enabled:     true,
		},
	}

	err := Save(cfg)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := os.ReadFile(Path())
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}

	if len(loaded) == 0 {
		t.Error("Saved config should not be empty")
	}

	loadedCfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loadedCfg.ClientID != cfg.ClientID {
		t.Errorf("ClientID = %s, want %s", loadedCfg.ClientID, cfg.ClientID)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir", ".whooper")

	origDir := dirFunc
	origPath := pathFunc

	dirFunc = func() string { return subDir }
	pathFunc = func() string { return filepath.Join(subDir, "config.yaml") }

	defer func() { dirFunc = origDir; pathFunc = origPath }()

	cfg := &Config{ClientID: "test"}

	err := Save(cfg)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(subDir)
	if err != nil {
		t.Fatalf("Stat error = %v", err)
	}
	if !info.IsDir() {
		t.Errorf("Expected directory, got %v", info.Mode())
	}
}
