package cmd

import (
	"context"
	"fmt"

	"git.infra.hisao.org/hisao/whooper/internal/api"
	"git.infra.hisao.org/hisao/whooper/internal/auth"
	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	gosync "git.infra.hisao.org/hisao/whooper/internal/sync"
	"git.infra.hisao.org/hisao/whooper/internal/tui"
	"git.infra.hisao.org/hisao/whooper/internal/tui/views"
	"github.com/spf13/cobra"
)

type tuiSyncRunner interface {
	SyncAll() error
}

var (
	tuiOpenDB     = store.Open
	tuiDBPath     = config.DBPath
	tuiLoadConfig = config.Load
	tuiLoadToken  = auth.LoadToken
	tuiTokenPath  = config.TokenPath
	tuiNewClient  = api.NewClient
	tuiNewSyncer  = func(client *api.Client, db *store.DB) tuiSyncRunner {
		return gosync.New(client, db, nil)
	}
	tuiRunApp = tui.RunApp
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive TUI dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := tuiOpenDB(tuiDBPath())
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer db.Close()

		// Build sync function if credentials are available
		var syncFn func() error
		cfg, cfgErr := tuiLoadConfig()
		if cfgErr == nil && cfg.ClientID != "" {
			token, tokErr := tuiLoadToken(tuiTokenPath())
			if tokErr == nil {
				oauthCfg := auth.OAuthConfig(cfg)
				tokenSource := auth.PersistingTokenSource(
					tuiTokenPath(),
					oauthCfg.TokenSource(context.Background(), token),
				)
				client := tuiNewClient(tokenSource)
				syncFn = func() error {
					syncer := tuiNewSyncer(client, db)
					return syncer.SyncAll()
				}
			}
		}

		app := tui.NewApp(syncFn)

		dashboard := views.NewDashboard(db)
		recovery := views.NewRecovery(db)
		sleep := views.NewSleep(db)
		workouts := views.NewWorkouts(db)
		correlations := views.NewCorrelations(db)

		app.SetViews(&dashboard, &recovery, &sleep, &workouts, &correlations)

		return tuiRunApp(app)
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
