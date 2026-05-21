package cmd

import (
	"context"
	"fmt"

	"git.infra.hisao.org/hisao/whooper/internal/analysis"
	"git.infra.hisao.org/hisao/whooper/internal/api"
	"git.infra.hisao.org/hisao/whooper/internal/auth"
	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	gosync "git.infra.hisao.org/hisao/whooper/internal/sync"
	"github.com/spf13/cobra"
)

var (
	syncFull  bool
	syncSince string
	syncDebug bool
)

type syncRunner interface {
	SyncFrom(string) error
}

var (
	syncLoadConfig = config.Load
	syncLoadToken  = auth.LoadToken
	syncTokenPath  = config.TokenPath
	syncDBPath     = config.DBPath
	syncOpenDB     = store.Open
	syncNewClient  = api.NewClient
	syncNewSyncer  = func(client *api.Client, db *store.DB, progress gosync.ProgressFunc) syncRunner {
		return gosync.New(client, db, progress)
	}
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Fetch data from the Whoop API and store locally",
	RunE: func(cmd *cobra.Command, args []string) error {
		debugf := func(format string, args ...any) {
			if syncDebug {
				fmt.Fprintf(cmd.OutOrStdout(), "[debug] "+format+"\n", args...)
			}
		}

		cfg, err := syncLoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		debugf("config loaded client_id_configured=%t client_secret_configured=%t", cfg.ClientID != "", cfg.ClientSecret != "")

		token, err := syncLoadToken(syncTokenPath())
		if err != nil {
			return fmt.Errorf("load token: %w; run 'whooper login' first", err)
		}
		debugf("token loaded token_path=%s", syncTokenPath())

		oauthCfg := auth.OAuthConfig(cfg)
		tokenSource := auth.PersistingTokenSource(
			syncTokenPath(),
			oauthCfg.TokenSource(context.Background(), token),
		)

		client := syncNewClient(tokenSource)
		db, err := syncOpenDB(syncDBPath())
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer db.Close()
		debugf("database opened db_path=%s", syncDBPath())

		syncer := syncNewSyncer(client, db, func(entity string, count int) {
			fmt.Printf("  %s: %d records\n", entity, count)
		})

		start := ""
		switch {
		case syncFull:
			fmt.Println("Performing full re-sync from Whoop...")
			start = "full"
		case syncSince != "":
			fmt.Printf("Syncing data from %s...\n", syncSince)
			start = syncSince + "T00:00:00.000Z"
		default:
			fmt.Println("Syncing data from Whoop...")
		}
		debugf("sync start=%q full=%t since=%q", start, syncFull, syncSince)

		if err := syncer.SyncFrom(start); err != nil {
			debugf("sync failed error=%q", err.Error())
			return fmt.Errorf("sync: %w", err)
		}
		fmt.Println("Sync complete!")

		// Check alerts
		alerts := analysis.CheckAlerts(db, cfg)
		debugf("alerts evaluated count=%d", len(alerts))
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
	syncCmd.Flags().BoolVar(&syncFull, "full", false, "Perform a full re-sync (ignore incremental state)")
	syncCmd.Flags().StringVar(&syncSince, "since", "", "Sync from a specific date (YYYY-MM-DD)")
	syncCmd.Flags().BoolVar(&syncDebug, "debug", false, "Print debug diagnostics during sync")
	rootCmd.AddCommand(syncCmd)
}
