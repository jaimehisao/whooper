package cmd

import (
	"context"
	"fmt"
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/analysis"
	"git.infra.hisao.org/hisao/whooper/internal/api"
	"git.infra.hisao.org/hisao/whooper/internal/auth"
	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	gosync "git.infra.hisao.org/hisao/whooper/internal/sync"
	"github.com/spf13/cobra"
)

var (
	syncFull     bool
	syncSince    string
	syncDebug    bool
	syncLoop     bool
	syncInterval time.Duration
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
	syncSleep          = time.Sleep
	syncLoopIterations int
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Fetch data from the Whoop API and store locally",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSyncCommand(cmd)
	},
}

func runSyncCommand(cmd *cobra.Command) error {
	if err := validateSyncSince(syncSince); err != nil {
		return err
	}

	if !syncLoop {
		return runSyncOnce(cmd)
	}
	if syncInterval <= 0 {
		return fmt.Errorf("sync interval must be greater than zero")
	}

	iterations := 0
	for {
		if err := runSyncOnce(cmd); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Sync error: %v\n", err)
		}
		iterations++
		if syncLoopIterations > 0 && iterations >= syncLoopIterations {
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Next sync in %s...\n", syncInterval)
		syncSleep(syncInterval)
	}
}

func validateSyncSince(value string) error {
	if value == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return fmt.Errorf("invalid --since value %q: must be YYYY-MM-DD", value)
	}
	return nil
}

func runSyncOnce(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	debugf := func(format string, args ...any) {
		if syncDebug {
			fmt.Fprintf(out, "[debug] "+format+"\n", args...)
		}
	}

	cfg, err := syncLoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	debugf("config loaded client_id_configured=%t client_secret_configured=%t", cfg.ClientID != "", cfg.ClientSecret != "")

	token, err := syncLoadToken(syncTokenPath())
	if err != nil {
		return fmt.Errorf("load token: %w\nRun 'whooper login' first.", err)
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
		fmt.Fprintf(out, "  %s: %d records\n", entity, count)
	})

	start := ""
	switch {
	case syncFull:
		fmt.Fprintln(out, "Performing full re-sync from Whoop...")
		start = "full"
	case syncSince != "":
		fmt.Fprintf(out, "Syncing data from %s...\n", syncSince)
		start = syncSince + "T00:00:00.000Z"
	default:
		fmt.Fprintln(out, "Syncing data from Whoop...")
	}
	debugf("sync start=%q full=%t since=%q", start, syncFull, syncSince)

	if err := syncer.SyncFrom(start); err != nil {
		debugf("sync failed error=%q", err.Error())
		if api.IsUnauthorized(err) {
			return fmt.Errorf("sync: WHOOP rejected the saved token; run 'whooper login' again: %w", err)
		}
		return fmt.Errorf("sync: %w", err)
	}
	fmt.Fprintln(out, "Sync complete!")

	// Check alerts
	alerts := analysis.CheckAlerts(db, cfg)
	debugf("alerts evaluated count=%d", len(alerts))
	for _, a := range alerts {
		switch a.Level {
		case "critical":
			fmt.Fprintf(out, "  [!] %s\n", a.Message)
		default:
			fmt.Fprintf(out, "  [*] %s\n", a.Message)
		}
	}
	return nil
}

func init() {
	syncCmd.Flags().BoolVar(&syncFull, "full", false, "Perform a full re-sync (ignore incremental state)")
	syncCmd.Flags().StringVar(&syncSince, "since", "", "Sync from a specific date (YYYY-MM-DD)")
	syncCmd.Flags().BoolVar(&syncDebug, "debug", false, "Print debug diagnostics during sync")
	syncCmd.Flags().BoolVar(&syncLoop, "loop", false, "Keep syncing in the foreground on an interval")
	syncCmd.Flags().DurationVar(&syncInterval, "interval", 30*time.Minute, "Interval for --loop")
	rootCmd.AddCommand(syncCmd)
}
