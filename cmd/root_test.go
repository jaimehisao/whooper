package cmd

import (
	"testing"
)

func TestSetVersion(t *testing.T) {
	orig := rootCmd.Version
	SetVersion("custom-version")
	if rootCmd.Version != "custom-version" {
		t.Errorf("Version = %s, want custom-version", rootCmd.Version)
	}
	rootCmd.Version = orig
}

func TestRootCmdUse(t *testing.T) {
	if rootCmd.Use != "whooper" {
		t.Errorf("Use = %s, want whooper", rootCmd.Use)
	}
}

func TestRootCmdSilenceUsage(t *testing.T) {
	if !rootCmd.SilenceUsage {
		t.Error("SilenceUsage should be true")
	}
}

func TestRootCmdShort(t *testing.T) {
	if rootCmd.Short == "" {
		t.Error("Short should not be empty")
	}
}
