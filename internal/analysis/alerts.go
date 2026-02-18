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
	var alerts []Alert

	recoveries, err := db.GetRecoveryTrend(today, "")
	if err == nil && len(recoveries) > 0 {
		r := recoveries[0]
		if r.RecoveryScore < cfg.Alerts.LowRecovery {
			level := "warning"
			if r.RecoveryScore < cfg.Alerts.LowRecovery/2 {
				level = "critical"
			}
			alerts = append(alerts, Alert{
				Level:   level,
				Message: fmt.Sprintf("Low recovery: %.0f%% (threshold: %.0f%%)", r.RecoveryScore, cfg.Alerts.LowRecovery),
			})
		}
	}

	strains, err := db.GetStrainTrend(today, "")
	if err == nil && len(strains) > 0 {
		s := strains[0]
		if s.Strain > cfg.Alerts.HighStrain {
			alerts = append(alerts, Alert{
				Level:   "warning",
				Message: fmt.Sprintf("High strain: %.1f (threshold: %.0f)", s.Strain, cfg.Alerts.HighStrain),
			})
		}
	}

	return alerts
}
