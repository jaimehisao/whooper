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

func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".whooper")
}

func Path() string {
	return filepath.Join(Dir(), "config.yaml")
}

func DBPath() string {
	return filepath.Join(Dir(), "whooper.db")
}

func TokenPath() string {
	return filepath.Join(Dir(), "token.json")
}

func Load() (*Config, error) {
	cfg := &Config{
		RedirectURL: "http://localhost:8484/callback",
		Alerts: Alerts{
			LowRecovery: 33,
			HighStrain:  18,
			Enabled:     true,
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
		cfg.RedirectURL = "http://localhost:8484/callback"
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
