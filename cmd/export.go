package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
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
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export data as JSON or CSV",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := store.Open(config.DBPath())
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer db.Close()

		var data any
		switch exportEntity {
		case "cycles":
			data, err = db.ListCycles("", "")
		case "recoveries":
			data, err = db.ListRecoveries("", "")
		case "sleeps":
			data, err = db.ListSleeps("", "", false)
		case "workouts":
			data, err = db.ListWorkouts("", "")
		default:
			return fmt.Errorf("unknown entity %q (valid: cycles, recoveries, sleeps, workouts)", exportEntity)
		}
		if err != nil {
			return fmt.Errorf("query %s: %w", exportEntity, err)
		}

		w := os.Stdout
		if exportOutput != "" {
			f, err := os.Create(exportOutput)
			if err != nil {
				return err
			}
			defer f.Close()
			w = f
		}

		switch exportFormat {
		case "json":
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			return enc.Encode(data)
		case "csv":
			return writeCSV(w, exportEntity, data)
		default:
			return fmt.Errorf("unknown format %q (valid: json, csv)", exportFormat)
		}
	},
}

func writeCSV(w *os.File, entity string, data any) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	switch entity {
	case "recoveries":
		writer.Write([]string{"cycle_id", "date", "recovery_score", "hrv", "rhr", "spo2", "skin_temp"})
		for _, r := range data.([]models.Recovery) {
			rec := []string{fmt.Sprint(r.CycleID), r.CreatedAt}
			if r.Score != nil {
				rec = append(rec, fmt.Sprintf("%.1f", r.Score.RecoveryScore),
					fmt.Sprintf("%.1f", r.Score.HRVRmssd),
					fmt.Sprintf("%.1f", r.Score.RestingHeartRate),
					fmt.Sprintf("%.1f", r.Score.SpO2Percentage),
					fmt.Sprintf("%.1f", r.Score.SkinTempCelsius))
			} else {
				rec = append(rec, "", "", "", "", "")
			}
			writer.Write(rec)
		}
	case "workouts":
		writer.Write([]string{"id", "date", "sport_id", "sport", "strain", "avg_hr", "max_hr", "kilojoule", "distance_m"})
		for _, wo := range data.([]models.Workout) {
			sport := models.SportName[wo.SportID]
			rec := []string{fmt.Sprint(wo.ID), wo.Start, fmt.Sprint(wo.SportID), sport}
			if wo.Score != nil {
				rec = append(rec, fmt.Sprintf("%.1f", wo.Score.Strain),
					fmt.Sprint(wo.Score.AverageHeartRate),
					fmt.Sprint(wo.Score.MaxHeartRate),
					fmt.Sprintf("%.1f", wo.Score.Kilojoule),
					fmt.Sprintf("%.1f", wo.Score.DistanceMeter))
			} else {
				rec = append(rec, "", "", "", "", "")
			}
			writer.Write(rec)
		}
	case "sleeps":
		writer.Write([]string{"id", "start", "end", "nap", "performance_pct", "efficiency_pct", "respiratory_rate"})
		for _, s := range data.([]models.Sleep) {
			nap := "false"
			if s.Nap {
				nap = "true"
			}
			rec := []string{fmt.Sprint(s.ID), s.Start, s.End, nap}
			if s.Score != nil {
				rec = append(rec, fmt.Sprintf("%.1f", s.Score.SleepPerformancePct),
					fmt.Sprintf("%.1f", s.Score.SleepEfficiencyPct),
					fmt.Sprintf("%.1f", s.Score.RespiratoryRate))
			} else {
				rec = append(rec, "", "", "")
			}
			writer.Write(rec)
		}
	case "cycles":
		writer.Write([]string{"id", "start", "end", "strain", "kilojoule", "avg_hr", "max_hr"})
		for _, c := range data.([]models.Cycle) {
			rec := []string{fmt.Sprint(c.ID), c.Start, c.End}
			if c.Score != nil {
				rec = append(rec, fmt.Sprintf("%.1f", c.Score.Strain),
					fmt.Sprintf("%.1f", c.Score.Kilojoule),
					fmt.Sprint(c.Score.AverageHeartRate),
					fmt.Sprint(c.Score.MaxHeartRate))
			} else {
				rec = append(rec, "", "", "", "")
			}
			writer.Write(rec)
		}
	}
	return nil
}

func init() {
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "json", "Output format (json, csv)")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file (default: stdout)")
	exportCmd.Flags().StringVarP(&exportEntity, "entity", "e", "recoveries", "Entity to export (cycles, recoveries, sleeps, workouts)")
	rootCmd.AddCommand(exportCmd)
}
