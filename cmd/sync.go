package cmd

import (
	"fmt"

	"git.infra.hisao.org/hisao/whooper/internal/analysis"
	"git.infra.hisao.org/hisao/whooper/internal/api"
	"git.infra.hisao.org/hisao/whooper/internal/auth"
	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	gosync "git.infra.hisao.org/hisao/whooper/internal/sync"
	"golang.org/x/oauth2"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Fetch data from the Whoop API and store locally",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		token, err := auth.LoadToken(config.TokenPath())
		if err != nil {
			return fmt.Errorf("load token: %w\nRun 'whooper login' first.", err)
		}

		oauthCfg := auth.OAuthConfig(cfg)
		tokenSource := auth.PersistingTokenSource(
			config.TokenPath(),
			oauthCfg.TokenSource(oauth2.NoContext, token),
		)

		client := api.NewClient(tokenSource)
		db, err := store.Open(config.DBPath())
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer db.Close()

		syncer := gosync.New(client, db, func(entity string, count int) {
			fmt.Printf("  %s: %d records\n", entity, count)
		})

		fmt.Println("Syncing data from Whoop...")
		if err := syncer.SyncAll(); err != nil {
			return fmt.Errorf("sync: %w", err)
		}
		fmt.Println("Sync complete!")

		// Check alerts
		alerts := analysis.CheckAlerts(db, cfg)
		for _, a := range alerts {
			switch a.Level {
			case "critical":
				fmt.Printf("  [!] %s\n", a.Message)
			default:
				fmt.Printf("  [*] %s\n", a.Message)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
