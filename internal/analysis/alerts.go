package analysis

import (
	"fmt"
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/store"
)

type Alert struct {
	Level   string // "warning" or "critical"
	Message string
}

// CheckAlerts evaluates today's data against configured thresholds.
func CheckAlerts(db *store.DB, cfg *config.Config) []Alert {
	if !cfg.Alerts.Enabled {
		return nil
	}

	today := time.Now().UTC().Format("2006-01-02")

	var recovery, strain float64
	recoveries, err := db.GetRecoveryTrend(today, "")
	if err == nil && len(recoveries) > 0 {
		recovery = recoveries[0].RecoveryScore
	}

	strains, err := db.GetStrainTrend(today, "")
	if err == nil && len(strains) > 0 {
		strain = strains[0].Strain
	}

	return EvaluateAlerts(recovery, strain, cfg.Alerts.LowRecovery, cfg.Alerts.HighStrain)
}

// EvaluateAlerts checks recovery and strain values against thresholds without DB access.
func EvaluateAlerts(recovery, strain, lowRecoveryThreshold, highStrainThreshold float64) []Alert {
	var alerts []Alert
	if recovery < lowRecoveryThreshold {
		level := "warning"
		if recovery < lowRecoveryThreshold/2 {
			level = "critical"
		}
		alerts = append(alerts, Alert{
			Level:   level,
			Message: fmt.Sprintf("Low recovery: %.0f%% (threshold: %.0f%%)", recovery, lowRecoveryThreshold),
		})
	}
	if strain > highStrainThreshold {
		alerts = append(alerts, Alert{
			Level:   "warning",
			Message: fmt.Sprintf("High strain: %.1f (threshold: %.0f)", strain, highStrainThreshold),
		})
	}
	return alerts
}
