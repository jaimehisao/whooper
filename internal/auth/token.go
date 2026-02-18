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

// persistingTokenSource wraps an oauth2.TokenSource and saves every new token
// to disk so that refreshed tokens survive process restarts.
type persistingTokenSource struct {
	path string
	src  oauth2.TokenSource
	mu   sync.Mutex
}

// PersistingTokenSource returns a TokenSource that persists every token
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
	if err := SaveToken(p.path, token); err != nil {
		return nil, err
	}
	return token, nil
}
