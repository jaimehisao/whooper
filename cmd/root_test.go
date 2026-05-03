package cmd

import (
	"bytes"
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

func TestExecuteSuccess(t *testing.T) {
	var out bytes.Buffer
	origOut := rootCmd.OutOrStdout()
	origErr := rootCmd.ErrOrStderr()
	origArgs := rootCmd.Flags().Args()

	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--help"})
	defer func() {
		rootCmd.SetOut(origOut)
		rootCmd.SetErr(origErr)
		rootCmd.SetArgs(origArgs)
	}()

	Execute()

	if out.Len() == 0 {
		t.Fatal("Execute --help wrote no output")
	}
}
