package cmd

import (
	"encoding/json"
	"fmt"
	"io"

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
	Errors                 []string          `json:"errors,omitempty"`
}

var statusJSON bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show local configuration, database, and sync status",
	RunE: func(cmd *cobra.Command, args []string) error {
		report := buildStatusReport()
		if statusJSON {
			return writeStatusJSON(cmd.OutOrStdout(), report)
		}
		writeStatusText(cmd.OutOrStdout(), report)
		return nil
	},
}

func buildStatusReport() statusReport {
	return buildStatusReportWithOpenDB(store.Open)
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
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output status as JSON")
	rootCmd.AddCommand(statusCmd)
}
