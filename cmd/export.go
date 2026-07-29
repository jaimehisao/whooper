package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/models"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	"github.com/spf13/cobra"
)

var (
	exportFormat string
	exportOutput string
	exportEntity string
	exportFrom   string
	exportTo     string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export data as JSON or CSV",
	Long:  "Export data as JSON or CSV from the local SQLite cache, or from a remote Whooper HTTP API when remote-url / WHOOPER_REMOTE_URL is set. Remote export uses /api/recovery, /api/sleep, /api/strain, and /api/workouts view rows.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate YYYY-MM-DD flags once; remote /api/* expects date-only query
		// params, while local List* queries need RFC3339 bounds from DateBounds.
		if err := validateDateRange(exportFrom, exportTo); err != nil {
			return err
		}

		backend, remoteOK, err := resolveRemoteBackend()
		if err != nil {
			return err
		}
		if remoteOK {
			return runExportRemote(cmd, backend, exportFrom, exportTo)
		}
		from, to, err := exportDateBounds(exportFrom, exportTo)
		if err != nil {
			return err
		}
		return runExportLocal(cmd, from, to)
	},
}

func runExportLocal(cmd *cobra.Command, from, to string) error {
	db, err := store.OpenReadOnly(config.DBPath())
	if err != nil {
		return fmt.Errorf("open database: %w\nHint: run 'whooper sync' or 'whooper login' first to initialize the database", err)
	}
	defer db.Close()

	var data any
	switch exportEntity {
	case "cycles":
		data, err = db.ListCycles(from, to)
	case "recoveries":
		data, err = db.ListRecoveries(from, to)
	case "sleeps":
		data, err = db.ListSleeps(from, to, false)
	case "workouts":
		data, err = db.ListWorkouts(from, to)
	default:
		return fmt.Errorf("unknown entity %q (valid: cycles, recoveries, sleeps, workouts)", exportEntity)
	}
	if err != nil {
		return fmt.Errorf("query %s: %w", exportEntity, err)
	}

	isEmpty := false
	switch v := data.(type) {
	case []models.Cycle:
		isEmpty = len(v) == 0
	case []models.Recovery:
		isEmpty = len(v) == 0
	case []models.Sleep:
		isEmpty = len(v) == 0
	case []models.Workout:
		isEmpty = len(v) == 0
	}
	if isEmpty {
		fmt.Fprintf(cmd.ErrOrStderr(), "No %s found in the local database.\nHint: Run 'whooper sync' to fetch data from Whoop.\n", exportEntity)
	}

	return writeExportOutput(cmd, data, exportEntity)
}

func runExportRemote(cmd *cobra.Command, backend remoteBackend, from, to string) error {
	path, err := entityAPIPath(exportEntity)
	if err != nil {
		return err
	}
	q := remoteQuery(from, to, maxAPILimit)
	var rows []map[string]any
	if err := backend.Client.GetJSON(path, q, &rows); err != nil {
		return formatRemoteError(err)
	}
	if len(rows) == 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "No %s found from remote backend.\nHint: Run 'whooper sync' on the remote host to fetch data from Whoop.\n", exportEntity)
	}
	return writeExportOutput(cmd, rows, exportEntity)
}

func writeExportOutput(cmd *cobra.Command, data any, entity string) error {
	var out io.Writer = cmd.OutOrStdout()
	var file *os.File
	var err error
	if exportOutput != "" {
		file, err = os.Create(exportOutput)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer file.Close()
		out = file
	}

	switch exportFormat {
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	case "csv":
		if rows, ok := data.([]map[string]any); ok {
			return writeCSVMaps(out, rows)
		}
		return writeCSVData(out, entity, data)
	default:
		return fmt.Errorf("unknown format %q (valid: json, csv)", exportFormat)
	}
}

func init() {
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "json", "Output format (json, csv)")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file (default: stdout)")
	exportCmd.Flags().StringVarP(&exportEntity, "entity", "e", "recoveries", "Entity to export (cycles, recoveries, sleeps, workouts)")
	exportCmd.Flags().StringVar(&exportFrom, "from", "", "Start date (YYYY-MM-DD)")
	exportCmd.Flags().StringVar(&exportTo, "to", "", "End date (YYYY-MM-DD)")
	rootCmd.AddCommand(exportCmd)
}
