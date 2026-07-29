package config

import (
	"os"
	"strings"
)

// Env keys for remote CLI client mode.
const (
	EnvRemoteURL   = "WHOOPER_REMOTE_URL"
	EnvRemoteToken = "WHOOPER_REMOTE_TOKEN"
	// EnvServeToken is also accepted as a client token for convenience when the
	// same value is used to protect whooper serve --allow-remote.
	EnvServeToken = "WHOOPER_SERVE_TOKEN"
)

// ResolvedRemoteURL returns the remote backend base URL from env, then config.
// Empty means local SQLite mode.
func ResolvedRemoteURL(cfg *Config) string {
	if v := strings.TrimSpace(os.Getenv(EnvRemoteURL)); v != "" {
		return strings.TrimRight(v, "/")
	}
	if cfg == nil {
		return ""
	}
	return strings.TrimRight(strings.TrimSpace(cfg.RemoteURL), "/")
}

// ResolvedRemoteToken returns the remote bearer token from env overrides, then config.
// Order: WHOOPER_REMOTE_TOKEN, WHOOPER_SERVE_TOKEN, config.remote_token.
func ResolvedRemoteToken(cfg *Config) string {
	if v := os.Getenv(EnvRemoteToken); v != "" {
		return v
	}
	if v := os.Getenv(EnvServeToken); v != "" {
		return v
	}
	if cfg == nil {
		return ""
	}
	return cfg.RemoteToken
}

// RemoteConfigured reports whether read commands should use the remote HTTP API.
func RemoteConfigured(cfg *Config) bool {
	return ResolvedRemoteURL(cfg) != ""
}
