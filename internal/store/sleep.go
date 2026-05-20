package store

import (
	"git.infra.hisao.org/hisao/whooper/internal/models"
)

func (db *DB) SaveSleeps(sleeps []models.Sleep) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO sleep
		(id, user_id, created_at, updated_at, start, end, nap, score_state,
		 total_in_bed_time_milli, total_awake_time_milli, total_no_data_time_milli,
		 total_light_sleep_time_milli, total_slow_wave_sleep_time_milli, total_rem_sleep_time_milli,
		 sleep_cycle_count, disturbance_count,
		 baseline_sleep_needed_milli, need_from_sleep_debt_milli,
		 need_from_recent_strain_milli, need_from_recent_nap_milli,
		 respiratory_rate, sleep_performance_pct, sleep_consistency_pct, sleep_efficiency_pct)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, s := range sleeps {
		var ss models.SleepStageSummary
		var sn models.SleepNeeded
		var respRate, perfPct, consPct, effPct float64
		if s.Score != nil {
			ss = s.Score.StageSummary
			sn = s.Score.SleepNeeded
			respRate = s.Score.RespiratoryRate
			perfPct = s.Score.SleepPerformancePct
			consPct = s.Score.SleepConsistencyPct
			effPct = s.Score.SleepEfficiencyPct
		}
		if _, err := stmt.Exec(
			s.ID, s.UserID, s.CreatedAt, s.UpdatedAt, s.Start, s.End, s.Nap, s.ScoreState,
			ss.TotalInBedTimeMilli, ss.TotalAwakeTimeMilli, ss.TotalNoDataTimeMilli,
			ss.TotalLightSleepTimeMilli, ss.TotalSlowWaveSleepTimeMilli, ss.TotalRemSleepTimeMilli,
			ss.SleepCycleCount, ss.DisturbanceCount,
			sn.BaselineMilli, sn.NeedFromSleepDebtMilli,
			sn.NeedFromRecentStrainMilli, sn.NeedFromRecentNapMilli,
			respRate, perfPct, consPct, effPct,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) ListSleeps(from, to string, excludeNaps bool) ([]models.Sleep, error) {
	query := `SELECT id, user_id, created_at, updated_at, start, end, nap, score_state,
		total_in_bed_time_milli, total_awake_time_milli, total_no_data_time_milli,
		total_light_sleep_time_milli, total_slow_wave_sleep_time_milli, total_rem_sleep_time_milli,
		sleep_cycle_count, disturbance_count,
		baseline_sleep_needed_milli, need_from_sleep_debt_milli,
		need_from_recent_strain_milli, need_from_recent_nap_milli,
		respiratory_rate, sleep_performance_pct, sleep_consistency_pct, sleep_efficiency_pct
		FROM sleep WHERE 1=1`
	args := []any{}

	if from != "" {
		query += ` AND start >= ?`
		args = append(args, from)
	}
	if to != "" {
		query += ` AND start <= ?`
		args = append(args, to)
	}
	if excludeNaps {
		query += ` AND nap = 0`
	}
	query += ` ORDER BY start DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sleeps := []models.Sleep{}
	if from != "" && to == "" {
		sleeps = make([]models.Sleep, 0, 90)
	}

	for rows.Next() {
		var s models.Sleep
		var ss models.SleepStageSummary
		var sn models.SleepNeeded
		var respRate, perfPct, consPct, effPct float64
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.CreatedAt, &s.UpdatedAt, &s.Start, &s.End, &s.Nap, &s.ScoreState,
			&ss.TotalInBedTimeMilli, &ss.TotalAwakeTimeMilli, &ss.TotalNoDataTimeMilli,
			&ss.TotalLightSleepTimeMilli, &ss.TotalSlowWaveSleepTimeMilli, &ss.TotalRemSleepTimeMilli,
			&ss.SleepCycleCount, &ss.DisturbanceCount,
			&sn.BaselineMilli, &sn.NeedFromSleepDebtMilli,
			&sn.NeedFromRecentStrainMilli, &sn.NeedFromRecentNapMilli,
			&respRate, &perfPct, &consPct, &effPct,
		); err != nil {
			return nil, err
		}
		s.Score = &models.SleepScore{
			StageSummary:        ss,
			SleepNeeded:         sn,
			RespiratoryRate:     respRate,
			SleepPerformancePct: perfPct,
			SleepConsistencyPct: consPct,
			SleepEfficiencyPct:  effPct,
		}
		sleeps = append(sleeps, s)
	}
	return sleeps, rows.Err()
}
