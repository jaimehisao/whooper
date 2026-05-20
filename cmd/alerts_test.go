package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/config"
)

func setupAlertsEnv(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	config.SetTestPaths(tmpDir, filepath.Join(tmpDir, "config.yaml"), filepath.Join(tmpDir, "whooper.db"))
	return tmpDir
}

func TestAlertsCmd(t *testing.T) {
	_ = setupAlertsEnv(t)

	// Capture stdout/stderr and reset after test
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	// Initial status
	rootCmd.SetArgs([]string{"alerts", "status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute status: %v", err)
	}
	out := buf.String()
	// Default is enabled according to internal/config/config.go
	if !strings.Contains(out, "Alerts:        enabled") {
		t.Errorf("Expected enabled alerts by default, got:\n%s", out)
	}

	// Test disable
	buf.Reset()
	rootCmd.SetArgs([]string{"alerts", "disable"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute disable: %v", err)
	}
	if !strings.Contains(buf.String(), "Alerts disabled.") {
		t.Errorf("Expected 'Alerts disabled.', got: %q", buf.String())
	}

	// Verify status after disable
	buf.Reset()
	rootCmd.SetArgs([]string{"alerts", "status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute status: %v", err)
	}
	if !strings.Contains(buf.String(), "Alerts:        disabled") {
		t.Errorf("Expected disabled alerts, got:\n%s", buf.String())
	}

	// Test enable
	buf.Reset()
	rootCmd.SetArgs([]string{"alerts", "enable"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute enable: %v", err)
	}
	if !strings.Contains(buf.String(), "Alerts enabled.") {
		t.Errorf("Expected 'Alerts enabled.', got: %q", buf.String())
	}

	// Test set low-recovery
	buf.Reset()
	rootCmd.SetArgs([]string{"alerts", "set", "low-recovery", "40.5"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute set low-recovery: %v", err)
	}
	if !strings.Contains(buf.String(), "Set low-recovery to 40.5 successfully.") {
		t.Errorf("Expected success message, got: %q", buf.String())
	}

	// Verify status
	buf.Reset()
	rootCmd.SetArgs([]string{"alerts", "status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute status: %v", err)
	}
	if !strings.Contains(buf.String(), "Low Recovery:  40.5%") {
		t.Errorf("Expected Low Recovery 40.5%%, got:\n%s", buf.String())
	}

	// Test set high-strain
	buf.Reset()
	rootCmd.SetArgs([]string{"alerts", "set", "high-strain", "19"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute set high-strain: %v", err)
	}
	if !strings.Contains(buf.String(), "Set high-strain to 19.0 successfully.") {
		t.Errorf("Expected success message, got: %q", buf.String())
	}

	// Verify status
	buf.Reset()
	rootCmd.SetArgs([]string{"alerts", "status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute status: %v", err)
	}
	if !strings.Contains(buf.String(), "High Strain:   19.0") {
		t.Errorf("Expected High Strain 19.0, got:\n%s", buf.String())
	}
}

func TestAlertsCmdValidation(t *testing.T) {
	_ = setupAlertsEnv(t)

	// Capture output and silence errors to avoid noisy logs during expected failures
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
		rootCmd.SilenceErrors = false
		rootCmd.SilenceUsage = true // Restore default
	})

	tests := []struct {
		args    []string
		wantErr string
	}{
		{[]string{"alerts", "set", "low-recovery", "abc"}, "invalid value \"abc\": must be a number"},
		{[]string{"alerts", "set", "low-recovery", "-1"}, "low-recovery must be between 0 and 100"},
		{[]string{"alerts", "set", "low-recovery", "101"}, "low-recovery must be between 0 and 100"},
		{[]string{"alerts", "set", "high-strain", "22"}, "high-strain must be between 0 and 21"},
		{[]string{"alerts", "set", "unknown", "10"}, "unknown key \"unknown\""},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, "_"), func(t *testing.T) {
			buf.Reset()
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()
			if err == nil {
				t.Errorf("Expected error for args %v, got nil", tt.args)
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}
