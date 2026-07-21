package formatters

import (
	"math"
	"strings"
	"testing"

	"git.infra.hisao.org/hisao/whooper/internal/store"
)

func TestFormatCorrelationData_Empty(t *testing.T) {
	result := FormatCorrelationData(nil, "recovery", "hrv")
	if result.HasData {
		t.Error("expected HasData=false for nil data")
	}

	result = FormatCorrelationData([]store.CorrelationPoint{}, "recovery", "hrv")
	if result.HasData {
		t.Error("expected HasData=false for empty data")
	}
}

func TestFormatCorrelationData_WithData(t *testing.T) {
	points := []store.CorrelationPoint{
		{X: 75, Y: 45},
		{X: 80, Y: 50},
		{X: 65, Y: 40},
	}

	result := FormatCorrelationData(points, "recovery", "hrv")

	if !result.HasData {
		t.Error("expected HasData=true")
	}
	if result.XMetric != "recovery" {
		t.Errorf("XMetric = %s, want recovery", result.XMetric)
	}
	if result.YMetric != "hrv" {
		t.Errorf("YMetric = %s, want hrv", result.YMetric)
	}
	if len(result.Points) != 3 {
		t.Errorf("len(Points) = %d, want 3", len(result.Points))
	}
}

func TestValidateMetric(t *testing.T) {
	if !ValidateMetric("recovery") {
		t.Error("expected recovery to be valid")
	}
	if !ValidateMetric("hrv") {
		t.Error("expected hrv to be valid")
	}
	if !ValidateMetric("strain") {
		t.Error("expected strain to be valid")
	}
	if ValidateMetric("invalid") {
		t.Error("expected invalid to be invalid")
	}
}

func TestAvailableMetrics(t *testing.T) {
	metrics := AvailableMetrics()
	if len(metrics) != 6 {
		t.Errorf("len(AvailableMetrics()) = %d, want 6", len(metrics))
	}
}

func TestCorrelationDescription(t *testing.T) {
	desc := CorrelationDescription("recovery", "hrv", 0.8)
	if desc == "" {
		t.Error("expected non-empty description")
	}
	if !strings.Contains(desc, "r=0.80") {
		t.Errorf("positive description missing r value: %s", desc)
	}

	desc = CorrelationDescription("recovery", "hrv", -0.6)
	if desc == "" {
		t.Error("expected non-empty description")
	}
	if !strings.Contains(desc, "r=-0.60") {
		t.Errorf("negative description missing r value: %s", desc)
	}
	if strings.Contains(desc, "(r=-)") {
		t.Errorf("negative correlation still formats as bare dash: %s", desc)
	}

	desc = CorrelationDescription("recovery", "hrv", 0.2)
	if desc == "" {
		t.Error("expected non-empty description")
	}
}

func TestFormatCorrelation_EdgeCases(t *testing.T) {
	t.Run("NaN and Infinity", func(t *testing.T) {
		points := []store.CorrelationPoint{
			{X: math.NaN(), Y: 50},
			{X: 80, Y: math.Inf(1)},
		}
		result := FormatCorrelationData(points, "recovery", "hrv")
		if !result.HasData {
			t.Error("expected HasData=true")
		}
	})

	t.Run("Single Point", func(t *testing.T) {
		points := []store.CorrelationPoint{{X: 50, Y: 50}}
		result := FormatCorrelationData(points, "recovery", "hrv")
		if !result.HasData {
			t.Error("expected HasData=true")
		}
	})

	t.Run("Large Range", func(t *testing.T) {
		points := []store.CorrelationPoint{
			{X: 1, Y: 1},
			{X: 1e10, Y: 1e10},
		}
		result := FormatCorrelationData(points, "recovery", "hrv")
		if !result.HasData {
			t.Error("expected HasData=true")
		}
	})

	t.Run("Description NaN and Inf", func(t *testing.T) {
		// Ensure it doesn't panic
		_ = CorrelationDescription("recovery", "hrv", math.NaN())
		_ = CorrelationDescription("recovery", "hrv", math.Inf(1))
		_ = CorrelationDescription("recovery", "hrv", math.Inf(-1))
	})
}
