package cmd

import (
	"fmt"

	"git.infra.hisao.org/hisao/whooper/internal/auth"
	"git.infra.hisao.org/hisao/whooper/internal/config"
	"github.com/spf13/cobra"
)

var oauthFlowFunc = auth.RunOAuthFlow

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
		token, err := oauthFlowFunc(oauthCfg)
		if err != nil {
			return fmt.Errorf("oauth flow: %w", err)
		}

		if err := auth.SaveToken(config.TokenPath(), token); err != nil {
			return fmt.Errorf("save token: %w", err)
		}

		fmt.Println("Login successful! Token saved.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
