package formatters

import (
	"math"
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/store"
)

func TestFormatRecoveryData_Empty(t *testing.T) {
	result := FormatRecoveryData(nil, 7, 80)
	if result.HasData {
		t.Error("expected HasData=false for nil data")
	}
	if result.HasData {
		t.Error("expected HasData=false for empty data")
	}
}

func TestFormatRecoveryData_WithData(t *testing.T) {
	data := []store.RecoveryTrendPoint{
		{Date: "2024-01-01", RecoveryScore: 65, HRV: 40, RHR: 55},
		{Date: "2024-01-02", RecoveryScore: 75, HRV: 45, RHR: 52},
		{Date: "2024-01-03", RecoveryScore: 85, HRV: 50, RHR: 50},
	}

	result := FormatRecoveryData(data, 7, 80)

	if !result.HasData {
		t.Error("expected HasData=true")
	}
	if len(result.RecoveryScores) != 3 {
		t.Errorf("len(RecoveryScores) = %d, want 3", len(result.RecoveryScores))
	}
	if result.Latest == nil {
		t.Error("expected Latest to be set")
	}
	if result.Latest.RecoveryScore != 85 {
		t.Errorf("Latest.RecoveryScore = %v, want 85", result.Latest.RecoveryScore)
	}
}

func TestFormatRecoveryData_CalculatesMA(t *testing.T) {
	data := make([]store.RecoveryTrendPoint, 10)
	for i := range data {
		data[i] = store.RecoveryTrendPoint{
			Date:          "2024-01-01",
			RecoveryScore: float64(i * 10),
			HRV:           float64(i * 5),
			RHR:           float64(60 - i*2),
		}
	}

	result := FormatRecoveryData(data, 7, 80)

	if len(result.RecoveryMA) == 0 {
		t.Error("expected RecoveryMA to be calculated")
	}
	if len(result.HRVMA) == 0 {
		t.Error("expected HRVMA to be calculated")
	}
}

func TestFormatRecoverySparkline(t *testing.T) {
	data := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}

	result := FormatRecoverySparkline(data, 50)
	if result == "" {
		t.Error("expected non-empty sparkline")
	}

	// Small width
	result = FormatRecoverySparkline(data, 5)
	if len(result) == 0 {
		t.Error("expected sparkline even with small width")
	}

	// All same values
	result = FormatRecoverySparkline([]float64{50, 50, 50}, 10)
	if len(result) == 0 {
		t.Error("expected sparkline for same values")
	}
}

func TestMinMaxAverage_Single(t *testing.T) {
	data := []float64{42}
	if minFloat(data) != 42 {
		t.Error("min single failed")
	}
	if maxFloat(data) != 42 {
		t.Error("max single failed")
	}
	if averageFloat(data) != 42 {
		t.Error("average single failed")
	}
}

func TestFormatRecoverySparkline_Empty(t *testing.T) {
	result := FormatRecoverySparkline(nil, 50)
	if result != "" {
		t.Errorf("expected empty string for nil data, got %q", result)
	}

	result = FormatRecoverySparkline([]float64{}, 50)
	if result != "" {
		t.Errorf("expected empty string for empty data, got %q", result)
	}
}

func TestFormatRecoverySummary(t *testing.T) {
	data := []store.RecoveryTrendPoint{
		{Date: "2024-01-01", RecoveryScore: 65, HRV: 40, RHR: 55},
		{Date: "2024-01-02", RecoveryScore: 75, HRV: 45, RHR: 52},
		{Date: "2024-01-03", RecoveryScore: 85, HRV: 50, RHR: 50},
	}

	result := FormatRecoverySummary(data)
	if result == "" {
		t.Error("expected non-empty summary")
	}
}

func TestFormatRecoverySummary_Empty(t *testing.T) {
	result := FormatRecoverySummary(nil)
	if result != "" {
		t.Errorf("expected empty string for nil data, got %q", result)
	}

	result = FormatRecoverySummary([]store.RecoveryTrendPoint{})
	if result != "" {
		t.Errorf("expected empty string for empty data, got %q", result)
	}
}

func TestRecoveryDateRange(t *testing.T) {
	from, to := RecoveryDateRange(7)
	if from == "" {
		t.Error("expected non-empty from date")
	}
	if to == "" {
		t.Error("expected non-empty to date")
	}
	if from >= to {
		t.Errorf("from (%s) should be before to (%s)", from, to)
	}
}

func TestMinFloat(t *testing.T) {
	data := []float64{5, 2, 8, 1, 9}
	result := minFloat(data)
	if result != 1 {
		t.Errorf("minFloat() = %v, want 1", result)
	}

	// NaN handling (current implementation min is data[0], then compares)
	// If first is NaN, it might stay NaN or return something else.
	// We just want to ensure no panic.
	_ = minFloat([]float64{math.NaN(), 1, 2})
	_ = minFloat([]float64{1, math.NaN(), 2})
	_ = minFloat([]float64{math.Inf(1), math.Inf(-1)})
}

func TestMaxFloat(t *testing.T) {
	data := []float64{5, 2, 8, 1, 9}
	result := maxFloat(data)
	if result != 9 {
		t.Errorf("maxFloat() = %v, want 9", result)
	}

	_ = maxFloat([]float64{math.NaN(), 1, 2})
	_ = maxFloat([]float64{1, math.NaN(), 2})
	_ = maxFloat([]float64{math.Inf(1), math.Inf(-1)})
}

func TestAverageFloat(t *testing.T) {
	data := []float64{10, 20, 30}
	result := averageFloat(data)
	if result != 20 {
		t.Errorf("averageFloat() = %v, want 20", result)
	}

	result = averageFloat(nil)
	if result != 0 {
		t.Errorf("averageFloat(nil) = %v, want 0", result)
	}

	_ = averageFloat([]float64{math.NaN(), 10})
	_ = averageFloat([]float64{math.Inf(1), -math.Inf(1)})
}

func TestFormatRecovery_EdgeCases(t *testing.T) {
	t.Run("NaN and Infinity", func(t *testing.T) {
		data := []store.RecoveryTrendPoint{
			{RecoveryScore: math.NaN(), HRV: math.Inf(1), RHR: math.Inf(-1)},
			{RecoveryScore: 100, HRV: 50, RHR: 60},
		}
		result := FormatRecoveryData(data, 7, 80)
		if !result.HasData {
			t.Error("expected HasData=true")
		}

		summary := FormatRecoverySummary(data)
		if summary == "" {
			t.Error("expected non-empty summary for extreme values")
		}
	})

	t.Run("Large Range Sparkline", func(t *testing.T) {
		data := []float64{1, 1e10, 1e-10, 50, math.Inf(1)}
		spark := FormatRecoverySparkline(data, 20)
		if spark == "" {
			t.Error("expected sparkline for large range")
		}
	})

	t.Run("Invalid Timestamps", func(t *testing.T) {
		data := []store.RecoveryTrendPoint{
			{Date: "not-a-date", RecoveryScore: 50},
		}
		result := FormatRecoveryData(data, 7, 80)
		if result.Latest.Date != "not-a-date" {
			t.Errorf("expected date preservation, got %q", result.Latest.Date)
		}
	})
}

func TestExtractField(t *testing.T) {
	data := []store.RecoveryTrendPoint{
		{RecoveryScore: 65},
		{RecoveryScore: 75},
		{RecoveryScore: 85},
	}

	result := extractField(data, func(p store.RecoveryTrendPoint) float64 {
		return p.RecoveryScore
	})

	if len(result) != 3 {
		t.Errorf("len(result) = %d, want 3", len(result))
	}
	if result[0] != 65 || result[1] != 75 || result[2] != 85 {
		t.Errorf("result = %v, want [65, 75, 85]", result)
	}
}
