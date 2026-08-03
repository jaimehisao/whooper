package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "whooper",
	Short: "Whoop API data extractor & TUI dashboard",
	Long:  "Whooper syncs your Whoop health data and presents it in a rich terminal dashboard.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return tuiCmd.RunE(cmd, args)
	},
	SilenceUsage: true,
}

func SetVersion(v string) {
	rootCmd.Version = v
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		var ae *agentExitError
		if errors.As(err, &ae) {
			// Agent commands already wrote JSON to stdout; avoid duplicating the message.
			os.Exit(ae.code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
