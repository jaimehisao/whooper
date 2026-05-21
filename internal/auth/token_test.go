package auth

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

var testTime = time.Now()
var errAssert = errors.New("assert error")

func TestSaveAndLoadToken(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "token.json")

	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		TokenType:    "Bearer",
		RefreshToken: "test-refresh-token",
		Expiry:       testTime,
	}

	err := SaveToken(tokenPath, token)
	if err != nil {
		t.Fatalf("SaveToken failed: %v", err)
	}

	loaded, err := LoadToken(tokenPath)
	if err != nil {
		t.Fatalf("LoadToken failed: %v", err)
	}

	if loaded.AccessToken != token.AccessToken {
		t.Errorf("AccessToken mismatch: got %s, want %s", loaded.AccessToken, token.AccessToken)
	}
	if loaded.TokenType != token.TokenType {
		t.Errorf("TokenType mismatch: got %s, want %s", loaded.TokenType, token.TokenType)
	}
	if loaded.RefreshToken != token.RefreshToken {
		t.Errorf("RefreshToken mismatch: got %s, want %s", loaded.RefreshToken, token.RefreshToken)
	}
}

func TestLoadTokenNotExist(t *testing.T) {
	_, err := LoadToken("/nonexistent/path/token.json")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestPersistingTokenSource(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "token.json")

	token := &oauth2.Token{
		AccessToken: "initial-token",
	}

	src := &mockTokenSource{token: token}
	persisting := PersistingTokenSource(tokenPath, src)

	got, err := persisting.Token()
	if err != nil {
		t.Fatalf("Token() failed: %v", err)
	}

	if got.AccessToken != "initial-token" {
		t.Errorf("first Token() = %s, want initial-token", got.AccessToken)
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("Failed to read token file: %v", err)
	}
	if len(data) == 0 {
		t.Error("Token file should not be empty")
	}

	token.AccessToken = "refreshed-token"
	got, err = persisting.Token()
	if err != nil {
		t.Fatalf("Token() failed: %v", err)
	}

	if got.AccessToken != "refreshed-token" {
		t.Errorf("second Token() = %s, want refreshed-token", got.AccessToken)
	}
}

func TestPersistingTokenSourceNoRefresh(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "token.json")

	token := &oauth2.Token{
		AccessToken: "same-token",
	}

	src := &mockTokenSource{token: token}
	persisting := PersistingTokenSource(tokenPath, src)

	persisting.Token()
	persisting.Token()
	persisting.Token()

	_, err := os.Stat(tokenPath)
	if err != nil {
		t.Errorf("Token file should exist after multiple calls: %v", err)
	}
}

func TestPersistingTokenSourceError(t *testing.T) {
	src := &mockTokenSource{err: errAssert}
	persisting := PersistingTokenSource("/dev/null/token.json", src)

	_, err := persisting.Token()
	if err != errAssert {
		t.Errorf("Token() error = %v, want %v", err, errAssert)
	}
}

type mockTokenSource struct {
	token *oauth2.Token
	err   error
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.token, nil
}

func TestLoadTokenInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "invalid.json")
	os.WriteFile(tokenPath, []byte("not valid json"), 0o600)

	_, err := LoadToken(tokenPath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSaveTokenError(t *testing.T) {
	err := SaveToken("/nonexistent/dir/token.json", &oauth2.Token{})
	if err == nil {
		t.Error("expected error when saving to nonexistent directory")
	}
}

func TestPersistingTokenSource_SaveError(t *testing.T) {
	token := &oauth2.Token{AccessToken: "abc"}
	src := &mockTokenSource{token: token}
	// / is a directory, cannot write file to it
	persisting := PersistingTokenSource("/", src)

	got, err := persisting.Token()
	if err != nil {
		t.Fatalf("Token() error = %v, expected no error even if save fails", err)
	}
	if got.AccessToken != "abc" {
		t.Errorf("got %s, want abc", got.AccessToken)
	}
}
