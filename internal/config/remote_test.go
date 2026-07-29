package config

import (
	"testing"
)

func TestResolvedRemoteURLFromConfig(t *testing.T) {
	t.Setenv(EnvRemoteURL, "")
	cfg := &Config{RemoteURL: "http://example:9464/"}
	if got := ResolvedRemoteURL(cfg); got != "http://example:9464" {
		t.Fatalf("ResolvedRemoteURL = %q, want trimmed base", got)
	}
}

func TestResolvedRemoteURLEnvOverridesConfig(t *testing.T) {
	t.Setenv(EnvRemoteURL, "http://env-host:1234/")
	cfg := &Config{RemoteURL: "http://file-host:9464"}
	if got := ResolvedRemoteURL(cfg); got != "http://env-host:1234" {
		t.Fatalf("ResolvedRemoteURL = %q, want env value", got)
	}
}

func TestResolvedRemoteTokenOrder(t *testing.T) {
	cfg := &Config{RemoteToken: "from-file"}

	t.Setenv(EnvRemoteToken, "")
	t.Setenv(EnvServeToken, "")
	if got := ResolvedRemoteToken(cfg); got != "from-file" {
		t.Fatalf("token = %q, want from-file", got)
	}

	t.Setenv(EnvServeToken, "from-serve")
	if got := ResolvedRemoteToken(cfg); got != "from-serve" {
		t.Fatalf("token = %q, want WHOOPER_SERVE_TOKEN", got)
	}

	t.Setenv(EnvRemoteToken, "from-remote")
	if got := ResolvedRemoteToken(cfg); got != "from-remote" {
		t.Fatalf("token = %q, want WHOOPER_REMOTE_TOKEN", got)
	}
}

func TestRemoteConfigured(t *testing.T) {
	t.Setenv(EnvRemoteURL, "")
	if RemoteConfigured(&Config{}) {
		t.Fatal("empty config should not be remote-configured")
	}
	if !RemoteConfigured(&Config{RemoteURL: "http://x"}) {
		t.Fatal("remote_url should enable remote mode")
	}
	t.Setenv(EnvRemoteURL, "http://y")
	if !RemoteConfigured(&Config{}) {
		t.Fatal("env URL should enable remote mode")
	}
}

func TestLoadSaveRemoteFields(t *testing.T) {
	tmp := t.TempDir()
	SetTestPaths(tmp, tmp+"/config.yaml", tmp+"/whooper.db")
	cfg := &Config{
		ClientID:    "id",
		RemoteURL:   "http://backend:9464",
		RemoteToken: "secret-token",
	}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.RemoteURL != "http://backend:9464" {
		t.Errorf("RemoteURL = %q", loaded.RemoteURL)
	}
	if loaded.RemoteToken != "secret-token" {
		t.Errorf("RemoteToken = %q", loaded.RemoteToken)
	}
}
