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
// Missing today's recovery or strain does not produce an alert.
// When multiple scored rows exist for today, the latest is used.
// Database errors are returned so callers can surface them instead of
// silently treating failures as "no alerts".
func CheckAlerts(db *store.DB, cfg *config.Config) ([]Alert, error) {
	if !cfg.Alerts.Enabled {
		return nil, nil
	}

	today := time.Now().UTC().Format("2006-01-02")

	var alerts []Alert
	recoveries, err := db.GetRecoveryTrend(today, today)
	if err != nil {
		return nil, fmt.Errorf("recovery trend: %w", err)
	}
	if len(recoveries) > 0 {
		// GetRecoveryTrend returns one row per day ordered ascending; with a
		// single-day window that is today's latest scored recovery.
		latest := recoveries[len(recoveries)-1]
		alerts = append(alerts, evaluateRecoveryAlert(latest.RecoveryScore, cfg.Alerts.LowRecovery)...)
	}

	strains, err := db.GetStrainTrend(today, today)
	if err != nil {
		return nil, fmt.Errorf("strain trend: %w", err)
	}
	if len(strains) > 0 {
		latest := strains[len(strains)-1]
		alerts = append(alerts, evaluateStrainAlert(latest.Strain, cfg.Alerts.HighStrain)...)
	}

	return alerts, nil
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
