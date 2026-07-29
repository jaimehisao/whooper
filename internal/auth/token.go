package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/oauth2"
)

// SaveToken writes an OAuth2 token to disk as JSON using an atomic rename.
func SaveToken(path string, token *oauth2.Token) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "token-*.json")
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
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

// LoadToken reads an OAuth2 token from a JSON file on disk.
func LoadToken(path string) (*oauth2.Token, error) {
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
	lastAccess string
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

	if token.AccessToken != p.lastAccess {
		p.lastAccess = token.AccessToken
		if err := SaveToken(p.path, token); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to persist refreshed token: %v\n", err)
		}
	}
	return token, nil
}
