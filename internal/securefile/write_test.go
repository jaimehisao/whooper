package securefile

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteCreatesFileWithMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission check on Windows")
	}

	path := filepath.Join(t.TempDir(), "secret.json")
	if err := Write(path, []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatalf("Write: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("mode = %o, want 0600", mode)
	}
}

func TestWriteTightensExistingLoosePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission check on Windows")
	}

	path := filepath.Join(t.TempDir(), "secret.json")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatalf("seed WriteFile: %v", err)
	}

	if err := Write(path, []byte("new"), 0o600); err != nil {
		t.Fatalf("Write: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("mode = %o, want 0600 after overwrite", mode)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "new" {
		t.Fatalf("contents = %q, want %q", data, "new")
	}
}
