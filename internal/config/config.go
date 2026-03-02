package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	RedirectURL  string `yaml:"redirect_url"`
	Alerts       Alerts `yaml:"alerts"`
}

type Alerts struct {
	LowRecovery float64 `yaml:"low_recovery"` // Alert when recovery below this (default 33)
	HighStrain  float64 `yaml:"high_strain"`  // Alert when strain above this (default 18)
	Enabled     bool    `yaml:"enabled"`
}

var dirFunc = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".whooper")
}

func Dir() string {
	return dirFunc()
}

var pathFunc = func() string {
	return filepath.Join(Dir(), "config.yaml")
}

func Path() string {
	return pathFunc()
}

var dbPathFunc = func() string {
	return filepath.Join(Dir(), "whooper.db")
}

func DBPath() string {
	return dbPathFunc()
}

func TokenPath() string {
	return filepath.Join(Dir(), "token.json")
}

func SetTestPaths(dir, cfgPath, dbPath string) {
	dirFunc = func() string { return dir }
	pathFunc = func() string { return cfgPath }
	dbPathFunc = func() string { return dbPath }
}

const defaultRedirectURL = "http://localhost:8484/callback"
const defaultLowRecovery = 33.0
const defaultHighStrain = 18.0
const defaultAlertsEnabled = true

func Load() (*Config, error) {
	cfg := &Config{
		RedirectURL: defaultRedirectURL,
		Alerts: Alerts{
			LowRecovery: defaultLowRecovery,
			HighStrain:  defaultHighStrain,
			Enabled:     defaultAlertsEnabled,
		},
	}
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.RedirectURL == "" {
		cfg.RedirectURL = defaultRedirectURL
	}
	return cfg, nil
}

func Save(cfg *Config) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), data, 0o600)
}
