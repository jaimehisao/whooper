package formatters

import (
	"fmt"
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/analysis"
	"git.infra.hisao.org/hisao/whooper/internal/store"
)

type RecoveryDisplayData struct {
	Range          int
	RecoveryScores []float64
	HRVValues      []float64
	RHRValues      []float64
	RecoveryMA     []float64
	HRVMA          []float64
	Latest         *RecoveryPoint
	Error          string
	HasData        bool
}

type RecoveryPoint struct {
	RecoveryScore float64
	HRV           float64
	RHR           float64
	Date          string
}

func FormatRecoveryData(data []store.RecoveryTrendPoint, rangeDays int, width int) RecoveryDisplayData {
	result := RecoveryDisplayData{
		Range:   rangeDays,
		HasData: len(data) > 0,
	}

	if len(data) == 0 {
		return result
	}

	var recScores, hrvValues, rhrValues []float64
	for _, d := range data {
		recScores = append(recScores, d.RecoveryScore)
		hrvValues = append(hrvValues, d.HRV)
		rhrValues = append(rhrValues, d.RHR)
	}

	result.RecoveryScores = recScores
	result.HRVValues = hrvValues
	result.RHRValues = rhrValues

	result.RecoveryMA = analysis.MovingAverage(recScores, 7)
	result.HRVMA = analysis.MovingAverage(hrvValues, 7)

	last := data[len(data)-1]
	result.Latest = &RecoveryPoint{
		RecoveryScore: last.RecoveryScore,
		HRV:           last.HRV,
		RHR:           last.RHR,
		Date:          last.Date,
	}

	return result
}

func FormatRecoverySparkline(data []float64, width int) string {
	if len(data) == 0 {
		return ""
	}
	w := width - 10
	if w < 20 {
		w = 20
	}
	if w > 60 {
		w = 60
	}
	return sparkline(data, w)
}

func FormatRecoverySummary(data []store.RecoveryTrendPoint) string {
	if len(data) == 0 {
		return ""
	}

	recent := data[len(data)-1]
	avgRecovery := averageFloat(extractField(data, func(p store.RecoveryTrendPoint) float64 {
		return p.RecoveryScore
	}))
	avgHRV := averageFloat(extractField(data, func(p store.RecoveryTrendPoint) float64 {
		return p.HRV
	}))

	return fmt.Sprintf("Current: %.0f%% | Avg: %.0f%% | Avg HRV: %.0f ms",
		recent.RecoveryScore, avgRecovery, avgHRV)
}

func RecoveryDateRange(days int) (string, string) {
	now := time.Now().UTC()
	to := now.Format("2006-01-02")
	from := now.Add(-time.Duration(days) * 24 * time.Hour).Format("2006-01-02")
	return from, to
}

func sparkline(data []float64, width int) string {
	if len(data) == 0 || width <= 0 {
		return ""
	}
	minVal := minFloat(data)
	maxVal := maxFloat(data)
	rangeVal := maxVal - minVal
	if rangeVal == 0 {
		rangeVal = 1
	}

	blocks := []rune(" ▁▂▃▄▅▆▇█")
	numBlocks := len(blocks) - 1

	result := make([]rune, 0, width)
	for i := 0; i < width; i++ {
		idx := int(float64(i) * float64(len(data)-1) / float64(width))
		if idx >= len(data) {
			idx = len(data) - 1
		}
		normalized := (data[idx] - minVal) / rangeVal
		blockIdx := int(normalized * float64(numBlocks))
		if blockIdx < 0 {
			blockIdx = 0
		}
		if blockIdx > numBlocks {
			blockIdx = numBlocks
		}
		result = append(result, blocks[blockIdx])
	}
	return string(result)
}

func minFloat(data []float64) float64 {
	min := data[0]
	for _, v := range data[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func maxFloat(data []float64) float64 {
	max := data[0]
	for _, v := range data[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func averageFloat(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func extractField[T any](slice []T, fn func(T) float64) []float64 {
	result := make([]float64, len(slice))
	for i, v := range slice {
		result[i] = fn(v)
	}
	return result
}
