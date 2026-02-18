package cmd

import (
	"fmt"

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
		masked := cfg.ClientSecret
		if len(masked) > 4 {
			masked = "****" + masked[len(masked)-4:]
		}
		fmt.Printf("Config file: %s\n", config.Path())
		fmt.Printf("  client_id:     %s\n", cfg.ClientID)
		fmt.Printf("  client_secret: %s\n", masked)
		fmt.Printf("  redirect_url:  %s\n", cfg.RedirectURL)
		fmt.Printf("  database:      %s\n", config.DBPath())
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long:  "Keys: client-id, client-secret, redirect-url",
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
			cfg.ClientSecret = args[1]
		case "redirect-url":
			cfg.RedirectURL = args[1]
		default:
			return fmt.Errorf("unknown key %q (valid: client-id, client-secret, redirect-url)", args[0])
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("Set %s successfully.\n", args[0])
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}
