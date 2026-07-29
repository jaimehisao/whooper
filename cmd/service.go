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
Prometheus and Grafana.

Non-loopback binds require --allow-remote and --token (or WHOOPER_SERVE_TOKEN).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateSyncSince(serviceSince); err != nil {
			return err
		}
		if serviceInterval <= 0 {
			return fmt.Errorf("service interval must be greater than zero")
		}

		token := resolveServeToken()
		if err := validateServeBind(serviceAddr, serveAllowRemote, token); err != nil {
			return err
		}

		// Map service flags to sync globals for runSyncOnce
		origFull, origSince, origDebug := syncFull, syncSince, syncDebug
		syncFull = serviceFull
		syncSince = serviceSince
		syncDebug = serviceDebug
		defer func() {
			syncFull, syncSince, syncDebug = origFull, origSince, origDebug
		}()

		handler := bearerAuthMiddleware(token, newServeHandler(buildServeStatusReport))
		errCh := make(chan error, 1)
		ready := make(chan struct{})
		// Capture locals so the listen goroutine does not race with test teardown
		// that swaps package-level serveListenAndServe / serviceAddr.
		addr := serviceAddr
		listen := serveListenAndServe
		go func() {
			err := listen(addr, handler, func() {
				close(ready)
			})
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("HTTP server error: %w", err)
			}
		}()

		select {
		case <-ready:
			// Print from the main goroutine so we do not race with sync progress
			// writes to the same command stdout (bytes.Buffer in tests).
			cmdFprintf(cmd, "Listening on http://%s\n", addr)
		case err := <-errCh:
			return err
		}

		iterations := 0
		for {
			if err := runSyncOnce(cmd); err != nil {
				cmdFprintf(cmd, "Sync error: %v\n", err)
			}

			iterations++
			if syncLoopIterations > 0 && iterations >= syncLoopIterations {
				break
			}

			select {
			case err := <-errCh:
				return err
			default:
			}

			cmdFprintf(cmd, "Next sync in %s...\n", serviceInterval)
			syncSleep(serviceInterval)

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
	serviceCmd.Flags().BoolVar(&serveAllowRemote, "allow-remote", false, "Allow binding to non-loopback addresses")
	serviceCmd.Flags().StringVar(&serveToken, "token", "", "Bearer token required when --allow-remote (or WHOOPER_SERVE_TOKEN)")
	rootCmd.AddCommand(serviceCmd)
}
