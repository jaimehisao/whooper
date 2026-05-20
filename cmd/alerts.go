package cmd

import (
	"fmt"
	"strconv"

	"git.infra.hisao.org/hisao/whooper/internal/config"
	"github.com/spf13/cobra"
)

var alertsCmd = &cobra.Command{
	Use:   "alerts",
	Short: "Manage alert configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		return alertsStatusCmd.RunE(cmd, args)
	},
}

var alertsStatusCmd = &cobra.Command{
	Use:     "status",
	Aliases: []string{"show"},
	Short:   "Show current alert configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		enabledStr := "disabled"
		if cfg.Alerts.Enabled {
			enabledStr = "enabled"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Alerts:        %s\n", enabledStr)
		fmt.Fprintf(cmd.OutOrStdout(), "Low Recovery:  %.1f%%\n", cfg.Alerts.LowRecovery)
		fmt.Fprintf(cmd.OutOrStdout(), "High Strain:   %.1f\n", cfg.Alerts.HighStrain)
		return nil
	},
}

var alertsEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable alerts",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		cfg.Alerts.Enabled = true
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Alerts enabled.")
		return nil
	},
}

var alertsDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable alerts",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		cfg.Alerts.Enabled = false
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Alerts disabled.")
		return nil
	},
}

var alertsSetCmd = &cobra.Command{
	Use:                "set <key> <value>",
	Short:              "Set an alert threshold",
	Long:               "Keys: low-recovery, high-strain",
	Args:               cobra.ExactArgs(2),
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		val, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return fmt.Errorf("invalid value %q: must be a number", args[1])
		}

		switch args[0] {
		case "low-recovery":
			if val < 0 || val > 100 {
				return fmt.Errorf("low-recovery must be between 0 and 100")
			}
			cfg.Alerts.LowRecovery = val
		case "high-strain":
			if val < 0 || val > 21 {
				return fmt.Errorf("high-strain must be between 0 and 21")
			}
			cfg.Alerts.HighStrain = val
		default:
			return fmt.Errorf("unknown key %q (valid: low-recovery, high-strain)", args[0])
		}

		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Set %s to %.1f successfully.\n", args[0], val)
		return nil
	},
}

func init() {
	alertsCmd.AddCommand(alertsStatusCmd)
	alertsCmd.AddCommand(alertsEnableCmd)
	alertsCmd.AddCommand(alertsDisableCmd)
	alertsCmd.AddCommand(alertsSetCmd)
	rootCmd.AddCommand(alertsCmd)
}
