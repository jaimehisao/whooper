package cmd

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"

	"git.infra.hisao.org/hisao/whooper/internal/models"
)

// writeCSVMaps writes remote API view rows as CSV (header = sorted keys of first row).
func writeCSVMaps(w io.Writer, rows []map[string]any) error {
	writer := csv.NewWriter(w)
	if len(rows) == 0 {
		writer.Flush()
		return writer.Error()
	}
	keys := make([]string, 0, len(rows[0]))
	for k := range rows[0] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if err := writer.Write(keys); err != nil {
		return err
	}
	for _, row := range rows {
		rec := make([]string, len(keys))
		for i, k := range keys {
			rec[i] = fmt.Sprint(row[k])
		}
		if err := writer.Write(rec); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func writeCSVData(w io.Writer, entity string, data any) error {
	writer := csv.NewWriter(w)

	write := func(rec []string) error {
		return writer.Write(rec)
	}

	switch entity {
	case "recoveries":
		if err := write([]string{"cycle_id", "date", "recovery_score", "hrv", "rhr", "spo2", "skin_temp"}); err != nil {
			return err
		}
		for _, r := range data.([]models.Recovery) {
			rec := []string{fmt.Sprint(r.CycleID), r.CreatedAt}
			if r.Score != nil {
				rec = append(rec, fmt.Sprintf("%.1f", r.Score.RecoveryScore),
					fmt.Sprintf("%.1f", r.Score.HRVRmssd),
					fmt.Sprintf("%.1f", r.Score.RestingHeartRate),
					fmt.Sprintf("%.1f", r.Score.SpO2OrZero()),
					fmt.Sprintf("%.1f", r.Score.SkinTempOrZero()))
			} else {
				rec = append(rec, "", "", "", "", "")
			}
			if err := write(rec); err != nil {
				return err
			}
		}
	case "workouts":
		if err := write([]string{"id", "date", "sport_id", "sport", "strain", "avg_hr", "max_hr", "kilojoule", "distance_m"}); err != nil {
			return err
		}
		for _, wo := range data.([]models.Workout) {
			sport := models.SportName[wo.SportID]
			rec := []string{fmt.Sprint(wo.ID), wo.Start, fmt.Sprint(wo.SportID), sport}
			if wo.Score != nil {
				rec = append(rec, fmt.Sprintf("%.1f", wo.Score.Strain),
					fmt.Sprint(wo.Score.AverageHeartRate),
					fmt.Sprint(wo.Score.MaxHeartRate),
					fmt.Sprintf("%.1f", wo.Score.Kilojoule),
					fmt.Sprintf("%.1f", wo.Score.DistanceOrZero()))
			} else {
				rec = append(rec, "", "", "", "", "")
			}
			if err := write(rec); err != nil {
				return err
			}
		}
	case "sleeps":
		if err := write([]string{"id", "start", "end", "nap", "performance_pct", "efficiency_pct", "respiratory_rate"}); err != nil {
			return err
		}
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
			if err := write(rec); err != nil {
				return err
			}
		}
	case "cycles":
		if err := write([]string{"id", "start", "end", "strain", "kilojoule", "avg_hr", "max_hr"}); err != nil {
			return err
		}
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
			if err := write(rec); err != nil {
				return err
			}
		}
	}
	writer.Flush()
	return writer.Error()
}
