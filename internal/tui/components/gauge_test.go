package components

import (
	"strings"
	"testing"
)

func TestGauge_HighRecovery(t *testing.T) {
	result := Gauge(85, 30)
	if !strings.Contains(result, "85%") {
		t.Error("should display recovery percentage")
	}
	if !strings.Contains(result, "Recovery:") {
		t.Error("should contain Recovery label")
	}
}

func TestGauge_LowRecovery(t *testing.T) {
	result := Gauge(20, 30)
	if !strings.Contains(result, "20%") {
		t.Error("should display recovery percentage")
	}
}

func TestGauge_ZeroWidth(t *testing.T) {
	// Width 0 defaults to 30
	result := Gauge(50, 0)
	if result == "" {
		t.Error("should render with default width")
	}
}

func TestGauge_Over100(t *testing.T) {
	// Should clamp to 100%
	result := Gauge(120, 20)
	if result == "" {
		t.Error("should not be empty")
	}
}

func TestGauge_Negative(t *testing.T) {
	result := Gauge(-10, 20)
	if result == "" {
		t.Error("should not be empty")
	}
}
