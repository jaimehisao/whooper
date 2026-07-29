package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"git.infra.hisao.org/hisao/whooper/internal/auth"
	"git.infra.hisao.org/hisao/whooper/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or set configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Config file: %s\n", config.Path())
		fmt.Fprintf(cmd.OutOrStdout(), "  client_id:     %s\n", cfg.ClientID)
		fmt.Fprintf(cmd.OutOrStdout(), "  client_secret: %s\n", maskSecret(cfg.ClientSecret))
		fmt.Fprintf(cmd.OutOrStdout(), "  redirect_url:  %s\n", cfg.RedirectURL)
		fmt.Fprintf(cmd.OutOrStdout(), "  database:      %s\n", config.DBPath())
		// Effective remote settings (env overrides applied).
		remoteURL := config.ResolvedRemoteURL(cfg)
		remoteTok := config.ResolvedRemoteToken(cfg)
		fmt.Fprintf(cmd.OutOrStdout(), "  remote_url:    %s\n", remoteURL)
		if remoteTok != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  remote_token:  %s\n", maskSecret(remoteTok))
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  remote_token:  \n")
		}
		if remoteURL != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  remote_mode:   enabled (read commands use remote HTTP API)\n")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  remote_mode:   disabled (local SQLite)\n")
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long:  "Keys: client-id, client-secret, redirect-url, remote-url, remote-token. For client-secret or remote-token, pass '-' to read from stdin. Use empty remote-url to clear remote mode.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		switch args[0] {
		case "client-id":
			cfg.ClientID = args[1]
		case "client-secret":
			secret := args[1]
			if secret == "-" {
				fmt.Fprint(cmd.OutOrStdout(), "Enter client secret: ")
				line, err := bufio.NewReader(os.Stdin).ReadString('\n')
				if err != nil {
					return fmt.Errorf("read client secret: %w", err)
				}
				secret = strings.TrimSpace(line)
			}
			cfg.ClientSecret = secret
		case "redirect-url":
			if err := auth.ValidateRedirectURL(args[1]); err != nil {
				return fmt.Errorf("invalid redirect-url: %w", err)
			}
			cfg.RedirectURL = args[1]
		case "remote-url":
			cfg.RemoteURL = strings.TrimSpace(args[1])
		case "remote-token":
			token := args[1]
			if token == "-" {
				fmt.Fprint(cmd.OutOrStdout(), "Enter remote token: ")
				line, err := bufio.NewReader(os.Stdin).ReadString('\n')
				if err != nil {
					return fmt.Errorf("read remote token: %w", err)
				}
				token = strings.TrimSpace(line)
			}
			cfg.RemoteToken = token
		default:
			return fmt.Errorf("unknown key %q (valid: client-id, client-secret, redirect-url, remote-url, remote-token)", args[0])
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Set %s successfully.\n", args[0])
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}
