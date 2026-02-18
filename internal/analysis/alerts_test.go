package analysis

import (
	"testing"
)

func TestEvaluateAlerts_LowRecovery(t *testing.T) {
	alerts := EvaluateAlerts(10, 10, 33, 18)
	found := false
	for _, a := range alerts {
		if a.Level == "critical" {
			found = true
		}
	}
	if !found {
		t.Error("expected critical alert for recovery below half threshold")
	}
}

func TestEvaluateAlerts_WarningRecovery(t *testing.T) {
	alerts := EvaluateAlerts(25, 10, 33, 18)
	found := false
	for _, a := range alerts {
		if a.Level == "warning" {
			found = true
		}
	}
	if !found {
		t.Error("expected warning alert for low recovery")
	}
}

func TestEvaluateAlerts_HighStrain(t *testing.T) {
	alerts := EvaluateAlerts(80, 20, 33, 18)
	found := false
	for _, a := range alerts {
		if a.Level == "warning" {
			found = true
		}
	}
	if !found {
		t.Error("expected warning alert for high strain")
	}
}

func TestEvaluateAlerts_NoAlerts(t *testing.T) {
	alerts := EvaluateAlerts(80, 10, 33, 18)
	if len(alerts) != 0 {
		t.Errorf("expected no alerts, got %d", len(alerts))
	}
}

func TestEvaluateAlerts_BothAlerts(t *testing.T) {
	alerts := EvaluateAlerts(20, 20, 33, 18)
	if len(alerts) != 2 {
		t.Errorf("expected 2 alerts, got %d", len(alerts))
	}
}
