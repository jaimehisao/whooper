package formatters

import (
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

	desc = CorrelationDescription("recovery", "hrv", -0.6)
	if desc == "" {
		t.Error("expected non-empty description")
	}

	desc = CorrelationDescription("recovery", "hrv", 0.2)
	if desc == "" {
		t.Error("expected non-empty description")
	}
}
