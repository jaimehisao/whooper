package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"git.infra.hisao.org/hisao/whooper/internal/analysis"
	"git.infra.hisao.org/hisao/whooper/internal/auth"
	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	"github.com/spf13/cobra"
)

type statusReport struct {
	ConfigPath             string            `json:"config_path"`
	DBPath                 string            `json:"db_path"`
	TokenPath              string            `json:"token_path"`
	ClientIDConfigured     bool              `json:"client_id_configured"`
	ClientSecretConfigured bool              `json:"client_secret_configured"`
	RedirectURL            string            `json:"redirect_url"`
	TokenPresent           bool              `json:"token_present"`
	DBOpen                 bool              `json:"db_open"`
	RecordCounts           map[string]int    `json:"record_counts,omitempty"`
	LastSync               map[string]string `json:"last_sync,omitempty"`
	LatestHealth           *healthReport     `json:"latest_health,omitempty"`
	AlertsEnabled          bool              `json:"alerts_enabled"`
	LowRecoveryThreshold   float64           `json:"low_recovery_threshold"`
	HighStrainThreshold    float64           `json:"high_strain_threshold"`
	AlertsFiring           int               `json:"alerts_firing"`
	AlertStates            map[string]int    `json:"alert_states,omitempty"`
	Errors                 []string          `json:"errors,omitempty"`
}

type healthReport struct {
	Values     map[string]float64 `json:"values,omitempty"`
	Timestamps map[string]string  `json:"timestamps,omitempty"`
}

var statusJSON bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show configuration, database, and sync status",
	Long:  "Show local configuration, database, and sync status. When remote-url / WHOOPER_REMOTE_URL is set, fetches /status from the remote Whooper backend instead of opening local SQLite.",
	RunE: func(cmd *cobra.Command, args []string) error {
		backend, remoteOK, err := resolveRemoteBackend()
		if err != nil {
			return err
		}
		var report statusReport
		if remoteOK {
			if err := backend.Client.GetJSON("/status", nil, &report); err != nil {
				return formatRemoteError(err)
			}
			// Annotate that this status came from remote (paths are the server's).
			if statusJSON {
				return writeStatusJSON(cmd.OutOrStdout(), report)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Remote backend: %s\n", backend.URL)
			writeStatusText(cmd.OutOrStdout(), report)
			return nil
		}
		report = buildStatusReport()
		if statusJSON {
			return writeStatusJSON(cmd.OutOrStdout(), report)
		}
		writeStatusText(cmd.OutOrStdout(), report)
		return nil
	},
}

func buildStatusReport() statusReport {
	return buildStatusReportWithOpenDB(store.OpenReadOnly)
}

func buildStatusReportWithOpenDB(openDB func(string) (*store.DB, error)) statusReport {
	report := statusReport{
		ConfigPath: config.Path(),
		DBPath:     config.DBPath(),
		TokenPath:  config.TokenPath(),
	}

	cfg, err := config.Load()
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("load config: %v", err))
	} else {
		report.ClientIDConfigured = cfg.ClientID != ""
		report.ClientSecretConfigured = cfg.ClientSecret != ""
		report.RedirectURL = cfg.RedirectURL
		report.AlertsEnabled = cfg.Alerts.Enabled
		report.LowRecoveryThreshold = cfg.Alerts.LowRecovery
		report.HighStrainThreshold = cfg.Alerts.HighStrain
	}

	if _, err := auth.LoadToken(config.TokenPath()); err == nil {
		report.TokenPresent = true
	}

	db, err := openDB(config.DBPath())
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("open database: %v", err))
		return report
	}
	defer db.Close()
	report.DBOpen = true

	if cfg != nil {
		alerts := analysis.CheckAlerts(db, cfg)
		report.AlertsFiring = len(alerts)
		report.AlertStates = map[string]int{
			"low_recovery": 0,
			"high_strain":  0,
		}
		for _, a := range alerts {
			if strings.Contains(strings.ToLower(a.Message), "recovery") {
				report.AlertStates["low_recovery"] = 1
			}
			if strings.Contains(strings.ToLower(a.Message), "strain") {
				report.AlertStates["high_strain"] = 1
			}
		}
	}

	report.RecordCounts = map[string]int{}
	for _, table := range statusEntities() {
		count, err := countStatusRows(db, table)
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("count %s: %v", table, err))
			continue
		}
		report.RecordCounts[table] = count
	}

	report.LastSync = map[string]string{}
	for _, entity := range statusEntities() {
		last, err := db.GetSyncState(entity)
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("sync state %s: %v", entity, err))
			continue
		}
		report.LastSync[entity] = last
	}

	latest, err := latestHealthReport(db)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("latest health: %v", err))
	} else if len(latest.Values) > 0 {
		report.LatestHealth = latest
	}

	return report
}

func countStatusRows(db *store.DB, entity string) (int, error) {
	table := map[string]string{
		"cycles":     "cycle",
		"recoveries": "recovery",
		"sleeps":     "sleep",
		"workouts":   "workout",
	}[entity]
	if table == "" {
		return 0, fmt.Errorf("unknown entity %q", entity)
	}
	var count int
	err := db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s`, table)).Scan(&count)
	return count, err
}

func statusEntities() []string {
	return []string{"cycles", "recoveries", "sleeps", "workouts"}
}

func latestHealthReport(db *store.DB) (*healthReport, error) {
	report := &healthReport{
		Values:     map[string]float64{},
		Timestamps: map[string]string{},
	}

	if err := addLatestRecoveryMetrics(db, report); err != nil {
		return nil, err
	}
	if err := addLatestSleepMetrics(db, report); err != nil {
		return nil, err
	}
	if err := addLatestStrainMetrics(db, report); err != nil {
		return nil, err
	}
	if err := addLatestWorkoutMetrics(db, report); err != nil {
		return nil, err
	}
	return report, nil
}

func addLatestRecoveryMetrics(db *store.DB, report *healthReport) error {
	var date string
	var recovery, hrv, rhr float64
	err := db.QueryRow(`SELECT date(created_at), recovery_score, hrv_rmssd, resting_heart_rate
		FROM recovery
		WHERE score_state = 'SCORED'
		ORDER BY created_at DESC
		LIMIT 1`).Scan(&date, &recovery, &hrv, &rhr)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("latest recovery: %w", err)
	}
	report.Values["recovery_score"] = recovery
	report.Values["hrv_rmssd"] = hrv
	report.Values["resting_heart_rate"] = rhr
	report.Timestamps["recovery"] = date
	return nil
}

func addLatestSleepMetrics(db *store.DB, report *healthReport) error {
	var date string
	var actualHours, needHours, debtHours, efficiency, performance, consistency float64
	err := db.QueryRow(`SELECT date(start),
			(total_in_bed_time_milli - total_awake_time_milli) / 3600000.0,
			(baseline_sleep_needed_milli + need_from_sleep_debt_milli + need_from_recent_strain_milli + need_from_recent_nap_milli) / 3600000.0,
			need_from_sleep_debt_milli / 3600000.0,
			sleep_efficiency_pct,
			sleep_performance_pct,
			sleep_consistency_pct
		FROM sleep
		WHERE nap = 0 AND score_state = 'SCORED'
		ORDER BY start DESC
		LIMIT 1`).Scan(&date, &actualHours, &needHours, &debtHours, &efficiency, &performance, &consistency)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("latest sleep: %w", err)
	}
	report.Values["sleep_actual_hours"] = actualHours
	report.Values["sleep_need_hours"] = needHours
	report.Values["sleep_need_gap_hours"] = actualHours - needHours
	report.Values["sleep_debt_hours"] = debtHours
	report.Values["sleep_efficiency_pct"] = efficiency
	report.Values["sleep_performance_pct"] = performance
	report.Values["sleep_consistency_pct"] = consistency
	report.Timestamps["sleep"] = date
	return nil
}

func addLatestStrainMetrics(db *store.DB, report *healthReport) error {
	var date string
	var strain float64
	err := db.QueryRow(`SELECT date(start), strain
		FROM cycle
		WHERE score_state = 'SCORED'
		ORDER BY start DESC
		LIMIT 1`).Scan(&date, &strain)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("latest strain: %w", err)
	}
	report.Values["day_strain"] = strain
	report.Timestamps["strain"] = date
	return nil
}

func addLatestWorkoutMetrics(db *store.DB, report *healthReport) error {
	var date string
	var strain, distanceKm, durationMinutes float64
	var avgHR, maxHR, sportID int
	err := db.QueryRow(`SELECT date(start), strain, average_heart_rate, max_heart_rate, distance_meter / 1000.0,
		(strftime('%s', end) - strftime('%s', start)) / 60.0, sport_id
		FROM workout
		WHERE score_state = 'SCORED'
		ORDER BY start DESC
		LIMIT 1`).Scan(&date, &strain, &avgHR, &maxHR, &distanceKm, &durationMinutes, &sportID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("latest workout: %w", err)
	}
	report.Values["workout_strain"] = strain
	report.Values["workout_average_heart_rate"] = float64(avgHR)
	report.Values["workout_max_heart_rate"] = float64(maxHR)
	report.Values["workout_distance_km"] = distanceKm
	report.Values["workout_duration_minutes"] = durationMinutes
	report.Values["workout_sport_id"] = float64(sportID)
	report.Timestamps["workout"] = date
	return nil
}

func writeStatusJSON(out io.Writer, report statusReport) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func writeStatusText(out io.Writer, report statusReport) {
	fmt.Fprintf(out, "Config: %s\n", report.ConfigPath)
	fmt.Fprintf(out, "Database: %s\n", report.DBPath)
	fmt.Fprintf(out, "Token: %s\n", report.TokenPath)
	fmt.Fprintf(out, "Client ID configured: %t\n", report.ClientIDConfigured)
	fmt.Fprintf(out, "Client secret configured: %t\n", report.ClientSecretConfigured)
	fmt.Fprintf(out, "Redirect URL: %s\n", report.RedirectURL)
	fmt.Fprintf(out, "Token present: %t\n", report.TokenPresent)
	fmt.Fprintf(out, "Database open: %t\n", report.DBOpen)

	if len(report.RecordCounts) > 0 {
		fmt.Fprintln(out, "Record counts:")
		for _, entity := range statusEntities() {
			fmt.Fprintf(out, "  %s: %d\n", entity, report.RecordCounts[entity])
		}
	}

	if len(report.LastSync) > 0 {
		fmt.Fprintln(out, "Last sync:")
		for _, entity := range statusEntities() {
			value := report.LastSync[entity]
			if value == "" {
				value = "never"
			}
			fmt.Fprintf(out, "  %s: %s\n", entity, value)
		}
	}

	for _, err := range report.Errors {
		fmt.Fprintf(out, "Error: %s\n", err)
	}

	if !report.ClientIDConfigured || !report.ClientSecretConfigured {
		fmt.Fprintln(out, "\nHint: Configure Whoop API credentials:")
		fmt.Fprintln(out, "  whooper config set client-id <id>")
		fmt.Fprintln(out, "  whooper config set client-secret <secret>")
	} else if !report.TokenPresent {
		fmt.Fprintln(out, "\nHint: Authenticate with Whoop:")
		fmt.Fprintln(out, "  whooper login")
	} else if !report.DBOpen {
		fmt.Fprintln(out, "\nHint: Initialize the database:")
		fmt.Fprintln(out, "  whooper sync")
	} else {
		totalRecords := 0
		for _, count := range report.RecordCounts {
			totalRecords += count
		}
		if totalRecords == 0 {
			fmt.Fprintln(out, "\nHint: Local database is empty. Sync data from Whoop:")
			fmt.Fprintln(out, "  whooper sync")
		}
	}
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output status as JSON")
	rootCmd.AddCommand(statusCmd)
}
