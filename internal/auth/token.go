package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/oauth2"
)

// SaveToken writes an OAuth2 token to disk as JSON with 0600 permissions.
// The write is atomic (temp file + rename) and serialized with an advisory
// file lock so concurrent whooper processes do not corrupt token.json.
func SaveToken(path string, token *oauth2.Token) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	unlock, err := lockTokenFile(path)
	if err != nil {
		return err
	}
	defer unlock()

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".token-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// LoadToken reads an OAuth2 token from a JSON file on disk.
// Files that are group/world-readable are rejected.
func LoadToken(path string) (*oauth2.Token, error) {
	unlock, err := lockTokenFile(path)
	if err == nil {
		defer unlock()
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		return nil, fmt.Errorf("token file %s has overly permissive mode %o (want 0600)", path, perm)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// persistingTokenSource wraps an oauth2.TokenSource and saves refreshed tokens
// to disk so they survive process restarts.
type persistingTokenSource struct {
	path       string
	src        oauth2.TokenSource
	mu         sync.Mutex
	lastAccess string // tracks last access token to detect refreshes
}

// PersistingTokenSource returns a TokenSource that persists refreshed tokens
// returned by src to the given file path.
func PersistingTokenSource(path string, src oauth2.TokenSource) oauth2.TokenSource {
	return &persistingTokenSource{path: path, src: src}
}

func (p *persistingTokenSource) Token() (*oauth2.Token, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	token, err := p.src.Token()
	if err != nil {
		return nil, err
	}

	// Only save when the token has actually been refreshed
	if token.AccessToken != p.lastAccess {
		p.lastAccess = token.AccessToken
		// We ignore save errors during refresh - it's better to have a fresh
		// token in memory than to fail because disk is full or read-only.
		_ = SaveToken(p.path, token)
	}
	return token, nil
}
