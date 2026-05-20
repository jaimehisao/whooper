package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/models"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	"github.com/spf13/cobra"
)

var summaryJSON bool

var summaryCmd = &cobra.Command{
	Use:     "summary",
	Aliases: []string{"inspect"},
	Short:   "Show latest health metrics and sync status",
	Long:    "Show latest recovery, HRV, sleep debt/gap, strain, latest workout context, and last sync state from the local SQLite cache.",
	RunE: func(cmd *cobra.Command, args []string) error {
		defer func() { summaryJSON = false }()

		db, err := store.Open(config.DBPath())
		if err != nil {
			return fmt.Errorf("open database: %w (hint: run 'whooper sync' or 'whooper login' first)", err)
		}
		defer db.Close()

		health, err := latestHealthReport(db)
		if err != nil {
			return err
		}

		syncStates := map[string]string{}
		for _, entity := range statusEntities() {
			last, err := db.GetSyncState(entity)
			if err != nil {
				continue
			}
			if last == "" {
				last = "never"
			}
			syncStates[entity] = last
		}

		if summaryJSON {
			return writeSummaryJSON(cmd.OutOrStdout(), health, syncStates)
		}

		if len(health.Values) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No local health data found.")
			fmt.Fprintln(cmd.OutOrStdout(), "Hint: Run 'whooper sync' to fetch data from Whoop.")
			fmt.Fprintln(cmd.OutOrStdout())
			writeSyncStates(cmd.OutOrStdout(), syncStates)
			return nil
		}

		writeSummaryText(cmd.OutOrStdout(), health, syncStates)
		return nil
	},
}

func writeSummaryJSON(out io.Writer, health *healthReport, syncStates map[string]string) error {
	report := struct {
		LatestHealth *healthReport     `json:"latest_health"`
		LastSync     map[string]string `json:"last_sync"`
	}{
		LatestHealth: health,
		LastSync:     syncStates,
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func writeSummaryText(out io.Writer, health *healthReport, syncStates map[string]string) {
	fmt.Fprintln(out, "Last Sync:")
	writeSyncStates(out, syncStates)
	fmt.Fprintln(out)

	if date, ok := health.Timestamps["recovery"]; ok {
		fmt.Fprintf(out, "Latest Health (%s):\n", date)
		fmt.Fprintf(out, "  Recovery: %.0f%%\n", health.Values["recovery_score"])
		fmt.Fprintf(out, "  HRV:      %.0f ms\n", health.Values["hrv_rmssd"])
		fmt.Fprintf(out, "  RHR:      %.0f bpm\n", health.Values["resting_heart_rate"])
	}

	if _, ok := health.Timestamps["sleep"]; ok {
		if _, ok := health.Timestamps["recovery"]; !ok {
			fmt.Fprintf(out, "Latest Sleep (%s):\n", health.Timestamps["sleep"])
		}
		fmt.Fprintf(out, "  Sleep:    %.1fh (Need: %.1fh, Gap: %+.1fh, Debt: %.1fh)\n",
			health.Values["sleep_actual_hours"],
			health.Values["sleep_need_hours"],
			health.Values["sleep_need_gap_hours"],
			health.Values["sleep_debt_hours"])
		fmt.Fprintf(out, "  Performance: %.0f%% (Efficiency: %.0f%%)\n",
			health.Values["sleep_performance_pct"],
			health.Values["sleep_efficiency_pct"])
	}

	if date, ok := health.Timestamps["strain"]; ok {
		fmt.Fprintf(out, "  Day Strain: %.1f (%s)\n", health.Values["day_strain"], date)
	}

	if date, ok := health.Timestamps["workout"]; ok {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "Latest Workout (%s):\n", date)
		sport := "Unknown"
		if sid, ok := health.Values["workout_sport_id"]; ok {
			if name, ok := models.SportName[int(sid)]; ok {
				sport = name
			}
		}
		fmt.Fprintf(out, "  %s: %.1f strain, %.0f min", sport, health.Values["workout_strain"], health.Values["workout_duration_minutes"])
		if dist := health.Values["workout_distance_km"]; dist > 0 {
			fmt.Fprintf(out, ", %.2f km", dist)
		}
		fmt.Fprintln(out)
		fmt.Fprintf(out, "  Avg HR: %.0f bpm (Max: %.0f bpm)\n",
			health.Values["workout_average_heart_rate"],
			health.Values["workout_max_heart_rate"])
	}
}

func writeSyncStates(out io.Writer, syncStates map[string]string) {
	for _, entity := range statusEntities() {
		fmt.Fprintf(out, "  %-12s %s\n", entity+":", syncStates[entity])
	}
}

func init() {
	summaryCmd.Flags().BoolVar(&summaryJSON, "json", false, "Output summary as JSON")
	rootCmd.AddCommand(summaryCmd)
}
