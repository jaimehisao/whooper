package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"git.infra.hisao.org/hisao/whooper/internal/securefile"
	"golang.org/x/oauth2"
)

// SaveToken writes an OAuth2 token to disk as JSON with 0600 permissions.
func SaveToken(path string, token *oauth2.Token) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return securefile.Write(path, data, 0o600)
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

	// Only save when the token has actually been refreshed.
	if token.AccessToken != p.lastAccess {
		p.lastAccess = token.AccessToken
		// Still return the refreshed token so the current request can proceed,
		// but surface persistence failures — a lost refresh token bricks auth
		// after restart when WHOOP rotates refresh tokens.
		if err := SaveToken(p.path, token); err != nil {
			fmt.Fprintf(os.Stderr, "whooper: warning: failed to persist refreshed OAuth token to %s: %v\n", p.path, err)
		}
	}
	return token, nil
}
