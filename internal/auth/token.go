package auth

import (
	"encoding/json"
	"os"
	"sync"

	"golang.org/x/oauth2"
)

// SaveToken writes an OAuth2 token to disk as JSON.
func SaveToken(path string, token *oauth2.Token) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
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
	path        string
	src         oauth2.TokenSource
	mu          sync.Mutex
	lastAccess  string // tracks last access token to detect refreshes
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
