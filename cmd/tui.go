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
		cfg, err := tuiLoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if cfg.ClientID == "" || cfg.ClientSecret == "" {
			return fmt.Errorf(
				"setup required before launching the dashboard.\nRun: whooper config set client-id <id>\n     whooper config set client-secret <secret>\nThen run: whooper login",
			)
		}

		db, err := tuiOpenDB(tuiDBPath())
		if err != nil {
			return fmt.Errorf("open database: %w\nHint: run 'whooper sync' or 'whooper login' first to initialize the database", err)
		}
		defer db.Close()

		// Build sync function if a token is available.
		var syncFn func() error
		canSync := false
		token, tokErr := tuiLoadToken(tuiTokenPath())
		if tokErr == nil {
			canSync = true
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

		app := tui.NewApp(syncFn)

		dashboard := views.NewDashboard(db)
		dashboard.SetConfig(cfg)
		dashboard.SetCanSync(canSync)
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
