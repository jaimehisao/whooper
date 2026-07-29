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
// Uses the latest sample for the day (List* is ordered DESC), not a daily average.
func CheckAlerts(db *store.DB, cfg *config.Config) []Alert {
	if !cfg.Alerts.Enabled {
		return nil
	}

	today := time.Now().UTC().Format("2006-01-02")

	var alerts []Alert
	recoveries, err := db.ListRecoveries(today, today)
	if err == nil {
		for _, r := range recoveries {
			if r.Score != nil {
				alerts = append(alerts, evaluateRecoveryAlert(r.Score.RecoveryScore, cfg.Alerts.LowRecovery)...)
				break
			}
		}
	}

	cycles, err := db.ListCycles(today, today)
	if err == nil {
		for _, c := range cycles {
			if c.Score != nil {
				alerts = append(alerts, evaluateStrainAlert(c.Score.Strain, cfg.Alerts.HighStrain)...)
				break
			}
		}
	}

	return alerts
}

// EvaluateAlerts checks recovery and strain values against thresholds without DB access.
func EvaluateAlerts(recovery, strain, lowRecoveryThreshold, highStrainThreshold float64) []Alert {
	var alerts []Alert
	alerts = append(alerts, evaluateRecoveryAlert(recovery, lowRecoveryThreshold)...)
	alerts = append(alerts, evaluateStrainAlert(strain, highStrainThreshold)...)
	return alerts
}

func evaluateRecoveryAlert(recovery, lowRecoveryThreshold float64) []Alert {
	if recovery < lowRecoveryThreshold {
		level := "warning"
		if recovery < lowRecoveryThreshold/2 {
			level = "critical"
		}
		return []Alert{{
			Level:   level,
			Message: fmt.Sprintf("Low recovery: %.0f%% (threshold: %.0f%%)", recovery, lowRecoveryThreshold),
		}}
	}
	return nil
}

func evaluateStrainAlert(strain, highStrainThreshold float64) []Alert {
	if strain > highStrainThreshold {
		return []Alert{{
			Level:   "warning",
			Message: fmt.Sprintf("High strain: %.1f (threshold: %.0f)", strain, highStrainThreshold),
		}}
	}
	return nil
}
