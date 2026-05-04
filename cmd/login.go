package cmd

import (
	"fmt"

	"git.infra.hisao.org/hisao/whooper/internal/auth"
	"git.infra.hisao.org/hisao/whooper/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var (
	oauthFlowFunc  = auth.RunOAuthFlow
	saveTokenFunc  = auth.SaveToken
	loginNoBrowser bool
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the Whoop API via OAuth2",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if cfg.ClientID == "" || cfg.ClientSecret == "" {
			return fmt.Errorf("client_id and client_secret must be configured first.\nRun: whooper config set client-id <id>\n     whooper config set client-secret <secret>")
		}

		oauthCfg := auth.OAuthConfig(cfg)
		flow := oauthFlowFunc
		if loginNoBrowser {
			flow = func(cfg *oauth2.Config) (*oauth2.Token, error) {
				return auth.RunOAuthFlowWithBrowser(cfg, false)
			}
		}
		token, err := flow(oauthCfg)
		if err != nil {
			return fmt.Errorf("oauth flow: %w", err)
		}

		if err := saveTokenFunc(config.TokenPath(), token); err != nil {
			return fmt.Errorf("save token: %w", err)
		}

		fmt.Println("Login successful! Token saved.")
		return nil
	},
}

func init() {
	loginCmd.Flags().BoolVar(&loginNoBrowser, "no-browser", false, "Print the authorization URL without opening a browser")
	rootCmd.AddCommand(loginCmd)
}
