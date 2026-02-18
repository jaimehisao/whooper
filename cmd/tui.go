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

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive TUI dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := store.Open(config.DBPath())
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer db.Close()

		// Build sync function if credentials are available
		var syncFn func() error
		cfg, cfgErr := config.Load()
		if cfgErr == nil && cfg.ClientID != "" {
			token, tokErr := auth.LoadToken(config.TokenPath())
			if tokErr == nil {
				oauthCfg := auth.OAuthConfig(cfg)
				tokenSource := auth.PersistingTokenSource(
					config.TokenPath(),
					oauthCfg.TokenSource(context.Background(), token),
				)
				client := api.NewClient(tokenSource)
				syncFn = func() error {
					syncer := gosync.New(client, db, nil)
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

		return tui.RunApp(app)
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
