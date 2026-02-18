package auth

import (
	"git.infra.hisao.org/hisao/whooper/internal/config"
	"golang.org/x/oauth2"
)

var whoopEndpoint = oauth2.Endpoint{
	AuthURL:  "https://api.prod.whoop.com/oauth/oauth2/auth",
	TokenURL: "https://api.prod.whoop.com/oauth/oauth2/token",
}

func OAuthConfig(cfg *config.Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes: []string{
			"read:recovery",
			"read:cycles",
			"read:sleep",
			"read:workout",
			"read:profile",
			"read:body_measurement",
		},
		Endpoint: whoopEndpoint,
	}
}
