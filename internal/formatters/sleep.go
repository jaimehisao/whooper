package formatters

import (
	"fmt"
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/analysis"
	"git.infra.hisao.org/hisao/whooper/internal/store"
)

type SleepDisplayData struct {
	Range        int
	SleepHours   []float64
	Efficiency   []float64
	SleepMA      []float64
	EfficiencyMA []float64
	Latest       *SleepPoint
	HasData      bool
}

type SleepPoint struct {
	Hours      float64
	Efficiency float64
	Date       string
}

func FormatSleepData(data []store.SleepTrendPoint, rangeDays int, width int) SleepDisplayData {
	result := SleepDisplayData{
		Range:   rangeDays,
		HasData: len(data) > 0,
	}

	if len(data) == 0 {
		return result
	}

	var hours, efficiency []float64
	for _, d := range data {
		hours = append(hours, float64(d.DurationMilli)/3600000.0)
		efficiency = append(efficiency, d.EfficiencyPct)
	}

	result.SleepHours = hours
	result.Efficiency = efficiency
	result.SleepMA = analysis.MovingAverage(hours, 7)
	result.EfficiencyMA = analysis.MovingAverage(efficiency, 7)

	last := data[len(data)-1]
	result.Latest = &SleepPoint{
		Hours:      float64(last.DurationMilli) / 3600000.0,
		Efficiency: last.EfficiencyPct,
		Date:       last.Date,
	}

	return result
}

func FormatSleepSummary(data []store.SleepTrendPoint) string {
	if len(data) == 0 {
		return ""
	}

	recent := data[len(data)-1]
	avgHours := averageFloat(extractFieldSleepHours(data))
	avgEff := averageFloat(extractField(data, func(p store.SleepTrendPoint) float64 {
		return p.EfficiencyPct
	}))

	return fmt.Sprintf("Last Night: %.1fh | Avg: %.1fh | Efficiency: %.0f%%",
		float64(recent.DurationMilli)/3600000.0, avgHours, avgEff)
}

func extractFieldSleepHours(data []store.SleepTrendPoint) []float64 {
	result := make([]float64, len(data))
	for i, v := range data {
		result[i] = float64(v.DurationMilli) / 3600000.0
	}
	return result
}

func SleepDateRange(days int) (string, string) {
	now := time.Now().UTC()
	to := now.Format("2006-01-02")
	from := now.Add(-time.Duration(days) * 24 * time.Hour).Format("2006-01-02")
	return from, to
}

func FormatSleepStageBars(data []store.SleepTrendPoint) string {
	if len(data) == 0 {
		return ""
	}
	return fmt.Sprintf("Sleep data for %d days", len(data))
}
