package formatters

import (
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/store"
)

func TestFormatSleepData_Empty(t *testing.T) {
	result := FormatSleepData(nil, 7, 80)
	if result.HasData {
		t.Error("expected HasData=false for nil data")
	}

	result = FormatSleepData([]store.SleepTrendPoint{}, 7, 80)
	if result.HasData {
		t.Error("expected HasData=false for empty data")
	}
}

func TestFormatSleepData_WithData(t *testing.T) {
	data := []store.SleepTrendPoint{
		{Date: "2024-01-01", DurationMilli: 28800000, EfficiencyPct: 90},
		{Date: "2024-01-02", DurationMilli: 32400000, EfficiencyPct: 92},
		{Date: "2024-01-03", DurationMilli: 25200000, EfficiencyPct: 85},
	}

	result := FormatSleepData(data, 7, 80)

	if !result.HasData {
		t.Error("expected HasData=true")
	}
	if len(result.SleepHours) != 3 {
		t.Errorf("len(SleepHours) = %d, want 3", len(result.SleepHours))
	}
	if result.Latest == nil {
		t.Error("expected Latest to be set")
	}
	if result.Latest.Hours != 7.0 {
		t.Errorf("Latest.Hours = %v, want 7.0", result.Latest.Hours)
	}
}

func TestFormatSleepData_CalculatesMA(t *testing.T) {
	data := make([]store.SleepTrendPoint, 10)
	for i := range data {
		data[i] = store.SleepTrendPoint{
			Date:          "2024-01-01",
			DurationMilli: 28800000 + i*3600000,
			EfficiencyPct: float64(80 + i),
		}
	}

	result := FormatSleepData(data, 7, 80)

	if len(result.SleepMA) == 0 {
		t.Error("expected SleepMA to be calculated")
	}
	if len(result.EfficiencyMA) == 0 {
		t.Error("expected EfficiencyMA to be calculated")
	}
}

func TestFormatSleepSummary(t *testing.T) {
	data := []store.SleepTrendPoint{
		{Date: "2024-01-01", DurationMilli: 28800000, EfficiencyPct: 90},
		{Date: "2024-01-02", DurationMilli: 32400000, EfficiencyPct: 92},
	}

	result := FormatSleepSummary(data)
	if result == "" {
		t.Error("expected non-empty summary")
	}
}

func TestFormatSleepSummary_Empty(t *testing.T) {
	result := FormatSleepSummary(nil)
	if result != "" {
		t.Errorf("expected empty string for nil data, got %q", result)
	}

	result = FormatSleepSummary([]store.SleepTrendPoint{})
	if result != "" {
		t.Errorf("expected empty string for empty data, got %q", result)
	}
}

func TestFormatSleepStageBars(t *testing.T) {
	data := []store.SleepTrendPoint{
		{Date: "2024-01-01", DurationMilli: 28800000, EfficiencyPct: 90},
		{Date: "2024-01-02", DurationMilli: 25200000, EfficiencyPct: 85},
	}

	got := FormatSleepStageBars(data)
	want := "Sleep data for 2 days"
	if got != want {
		t.Fatalf("FormatSleepStageBars() = %q, want %q", got, want)
	}
}

func TestFormatSleepStageBars_Empty(t *testing.T) {
	if got := FormatSleepStageBars(nil); got != "" {
		t.Fatalf("FormatSleepStageBars(nil) = %q, want empty", got)
	}
}

func TestSleepDateRange(t *testing.T) {
	from, to := SleepDateRange(7)
	if from == "" || to == "" {
		t.Error("expected non-empty dates")
	}
	if from >= to {
		t.Errorf("from (%s) should be before to (%s)", from, to)
	}
}

func TestExtractFieldSleepHours(t *testing.T) {
	data := []store.SleepTrendPoint{
		{DurationMilli: 28800000},
		{DurationMilli: 32400000},
		{DurationMilli: 25200000},
	}

	result := extractFieldSleepHours(data)

	if len(result) != 3 {
		t.Errorf("len(result) = %d, want 3", len(result))
	}
	if result[0] != 8.0 {
		t.Errorf("result[0] = %v, want 8.0", result[0])
	}
}
