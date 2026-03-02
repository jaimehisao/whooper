package auth

import (
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/config"
)

func TestOAuthConfig(t *testing.T) {
	cfg := &config.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8484/callback",
	}

	oauthCfg := OAuthConfig(cfg)

	if oauthCfg.ClientID != cfg.ClientID {
		t.Errorf("ClientID = %s, want %s", oauthCfg.ClientID, cfg.ClientID)
	}
	if oauthCfg.ClientSecret != cfg.ClientSecret {
		t.Errorf("ClientSecret = %s, want %s", oauthCfg.ClientSecret, cfg.ClientSecret)
	}
	if oauthCfg.RedirectURL != cfg.RedirectURL {
		t.Errorf("RedirectURL = %s, want %s", oauthCfg.RedirectURL, cfg.RedirectURL)
	}

	expectedScopes := []string{
		"read:recovery",
		"read:cycles",
		"read:sleep",
		"read:workout",
		"read:profile",
		"read:body_measurement",
	}

	if len(oauthCfg.Scopes) != len(expectedScopes) {
		t.Errorf("Scopes length = %d, want %d", len(oauthCfg.Scopes), len(expectedScopes))
	}

	for i, scope := range expectedScopes {
		if i >= len(oauthCfg.Scopes) || oauthCfg.Scopes[i] != scope {
			t.Errorf("Scope[%d] = %s, want %s", i, oauthCfg.Scopes[i], scope)
		}
	}

	if oauthCfg.Endpoint.AuthURL != "https://api.prod.whoop.com/oauth/oauth2/auth" {
		t.Errorf("AuthURL = %s", oauthCfg.Endpoint.AuthURL)
	}
	if oauthCfg.Endpoint.TokenURL != "https://api.prod.whoop.com/oauth/oauth2/token" {
		t.Errorf("TokenURL = %s", oauthCfg.Endpoint.TokenURL)
	}
}
