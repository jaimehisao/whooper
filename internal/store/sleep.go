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

	stmt, err := tx.Prepare(`INSERT INTO sleep
		(id, user_id, created_at, updated_at, start, end, timezone_offset, nap, score_state,
		 total_in_bed_time_milli, total_awake_time_milli, total_no_data_time_milli,
		 total_light_sleep_time_milli, total_slow_wave_sleep_time_milli, total_rem_sleep_time_milli,
		 sleep_cycle_count, disturbance_count,
		 baseline_sleep_needed_milli, need_from_sleep_debt_milli,
		 need_from_recent_strain_milli, need_from_recent_nap_milli,
		 respiratory_rate, sleep_performance_pct, sleep_consistency_pct, sleep_efficiency_pct)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			user_id = excluded.user_id,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			start = excluded.start,
			end = excluded.end,
			timezone_offset = excluded.timezone_offset,
			nap = excluded.nap,
			score_state = excluded.score_state,
			total_in_bed_time_milli = CASE WHEN ? THEN excluded.total_in_bed_time_milli ELSE sleep.total_in_bed_time_milli END,
			total_awake_time_milli = CASE WHEN ? THEN excluded.total_awake_time_milli ELSE sleep.total_awake_time_milli END,
			total_no_data_time_milli = CASE WHEN ? THEN excluded.total_no_data_time_milli ELSE sleep.total_no_data_time_milli END,
			total_light_sleep_time_milli = CASE WHEN ? THEN excluded.total_light_sleep_time_milli ELSE sleep.total_light_sleep_time_milli END,
			total_slow_wave_sleep_time_milli = CASE WHEN ? THEN excluded.total_slow_wave_sleep_time_milli ELSE sleep.total_slow_wave_sleep_time_milli END,
			total_rem_sleep_time_milli = CASE WHEN ? THEN excluded.total_rem_sleep_time_milli ELSE sleep.total_rem_sleep_time_milli END,
			sleep_cycle_count = CASE WHEN ? THEN excluded.sleep_cycle_count ELSE sleep.sleep_cycle_count END,
			disturbance_count = CASE WHEN ? THEN excluded.disturbance_count ELSE sleep.disturbance_count END,
			baseline_sleep_needed_milli = CASE WHEN ? THEN excluded.baseline_sleep_needed_milli ELSE sleep.baseline_sleep_needed_milli END,
			need_from_sleep_debt_milli = CASE WHEN ? THEN excluded.need_from_sleep_debt_milli ELSE sleep.need_from_sleep_debt_milli END,
			need_from_recent_strain_milli = CASE WHEN ? THEN excluded.need_from_recent_strain_milli ELSE sleep.need_from_recent_strain_milli END,
			need_from_recent_nap_milli = CASE WHEN ? THEN excluded.need_from_recent_nap_milli ELSE sleep.need_from_recent_nap_milli END,
			respiratory_rate = CASE WHEN ? THEN excluded.respiratory_rate ELSE sleep.respiratory_rate END,
			sleep_performance_pct = CASE WHEN ? THEN excluded.sleep_performance_pct ELSE sleep.sleep_performance_pct END,
			sleep_consistency_pct = CASE WHEN ? THEN excluded.sleep_consistency_pct ELSE sleep.sleep_consistency_pct END,
			sleep_efficiency_pct = CASE WHEN ? THEN excluded.sleep_efficiency_pct ELSE sleep.sleep_efficiency_pct END`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, s := range sleeps {
		var ss models.SleepStageSummary
		var sn models.SleepNeeded
		var respRate, perfPct, consPct, effPct float64
		hasScore := s.Score != nil
		if hasScore {
			ss = s.Score.StageSummary
			sn = s.Score.SleepNeeded
			respRate = s.Score.RespiratoryRate
			perfPct = s.Score.SleepPerformancePct
			consPct = s.Score.SleepConsistencyPct
			effPct = s.Score.SleepEfficiencyPct
		}
		hs := boolToInt(hasScore)
		if _, err := stmt.Exec(
			s.ID, s.UserID, s.CreatedAt, s.UpdatedAt, s.Start, s.End, s.TimezoneOffset, s.Nap, s.ScoreState,
			ss.TotalInBedTimeMilli, ss.TotalAwakeTimeMilli, ss.TotalNoDataTimeMilli,
			ss.TotalLightSleepTimeMilli, ss.TotalSlowWaveSleepTimeMilli, ss.TotalRemSleepTimeMilli,
			ss.SleepCycleCount, ss.DisturbanceCount,
			sn.BaselineMilli, sn.NeedFromSleepDebtMilli,
			sn.NeedFromRecentStrainMilli, sn.NeedFromRecentNapMilli,
			respRate, perfPct, consPct, effPct,
			hs, hs, hs, hs, hs, hs, hs, hs, hs, hs, hs, hs, hs, hs, hs, hs,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) ListSleeps(from, to string, excludeNaps bool) ([]models.Sleep, error) {
	query := `SELECT id, user_id, created_at, updated_at, start, end, timezone_offset, nap, score_state,
		total_in_bed_time_milli, total_awake_time_milli, total_no_data_time_milli,
		total_light_sleep_time_milli, total_slow_wave_sleep_time_milli, total_rem_sleep_time_milli,
		sleep_cycle_count, disturbance_count,
		baseline_sleep_needed_milli, need_from_sleep_debt_milli,
		need_from_recent_strain_milli, need_from_recent_nap_milli,
		respiratory_rate, sleep_performance_pct, sleep_consistency_pct, sleep_efficiency_pct
		FROM sleep WHERE 1=1`
	args := []any{}
	var err error
	query, args, err = appendListBounds(query, args, "start", from, to)
	if err != nil {
		return nil, err
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
			&s.ID, &s.UserID, &s.CreatedAt, &s.UpdatedAt, &s.Start, &s.End, &s.TimezoneOffset, &s.Nap, &s.ScoreState,
			&ss.TotalInBedTimeMilli, &ss.TotalAwakeTimeMilli, &ss.TotalNoDataTimeMilli,
			&ss.TotalLightSleepTimeMilli, &ss.TotalSlowWaveSleepTimeMilli, &ss.TotalRemSleepTimeMilli,
			&ss.SleepCycleCount, &ss.DisturbanceCount,
			&sn.BaselineMilli, &sn.NeedFromSleepDebtMilli,
			&sn.NeedFromRecentStrainMilli, &sn.NeedFromRecentNapMilli,
			&respRate, &perfPct, &consPct, &effPct,
		); err != nil {
			return nil, err
		}
		if s.ScoreState == "SCORED" {
			s.Score = &models.SleepScore{
				StageSummary:        ss,
				SleepNeeded:         sn,
				RespiratoryRate:     respRate,
				SleepPerformancePct: perfPct,
				SleepConsistencyPct: consPct,
				SleepEfficiencyPct:  effPct,
			}
		}
		sleeps = append(sleeps, s)
	}
	return sleeps, rows.Err()
}
