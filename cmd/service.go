package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

var (
	serviceAddr     string
	serviceInterval time.Duration
	serviceSince    string
	serviceFull     bool
	serviceDebug    bool
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Run combined sync loop and HTTP server",
	Long: `Run a combined service that performs periodic data synchronization and
serves the observability HTTP API from a single process.

This is the recommended way to run Whooper as a background bridge for
Prometheus and Grafana.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateSyncSince(serviceSince); err != nil {
			return err
		}
		if serviceInterval <= 0 {
			return fmt.Errorf("service interval must be greater than zero")
		}

		// Map service flags to sync globals for runSyncOnce
		origFull, origSince, origDebug := syncFull, syncSince, syncDebug
		syncFull = serviceFull
		syncSince = serviceSince
		syncDebug = serviceDebug
		defer func() {
			syncFull, syncSince, syncDebug = origFull, origSince, origDebug
		}()

		handler := newServeHandler(buildServeStatusReport)
		errCh := make(chan error, 1)
		addr := serviceAddr
		listen := serveListenAndServe
		go func() {
			if err := listen(addr, handler); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("HTTP server error: %w", err)
			}
		}()
		// Print from the main goroutine so we do not race with sync progress
		// writes to the same command stdout (bytes.Buffer in tests).
		fmt.Fprintf(cmd.OutOrStdout(), "Listening on http://%s\n", addr)

		iterations := 0
		for {
			// Run sync immediately on startup and then on interval
			if err := runSyncOnce(cmd); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Sync error: %v\n", err)
			}

			iterations++
			if syncLoopIterations > 0 && iterations >= syncLoopIterations {
				break
			}

			// Check for server errors before sleeping
			select {
			case err := <-errCh:
				return err
			default:
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Next sync in %s...\n", serviceInterval)
			syncSleep(serviceInterval)

			// Check for server errors after sleeping
			select {
			case err := <-errCh:
				return err
			default:
			}
		}

		return nil
	},
}

func init() {
	serviceCmd.Flags().StringVar(&serviceAddr, "addr", defaultServeAddr, "Address to bind the observability server")
	serviceCmd.Flags().DurationVar(&serviceInterval, "interval", 30*time.Minute, "Interval for periodic sync")
	serviceCmd.Flags().StringVar(&serviceSince, "since", "", "Sync from a specific date (YYYY-MM-DD)")
	serviceCmd.Flags().BoolVar(&serviceFull, "full", false, "Perform a full re-sync (ignore incremental state)")
	serviceCmd.Flags().BoolVar(&serviceDebug, "debug", false, "Print debug diagnostics during sync")
	rootCmd.AddCommand(serviceCmd)
}
