package store

import (
	"database/sql"

	"git.infra.hisao.org/hisao/whooper/internal/models"
)

func (db *DB) SaveRecoveries(recoveries []models.Recovery) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO recovery
		(cycle_id, sleep_id, user_id, created_at, updated_at, score_state,
		 user_calibrating, recovery_score, resting_heart_rate, hrv_rmssd,
		 spo2_percentage, skin_temp_celsius)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cycle_id) DO UPDATE SET
			sleep_id = excluded.sleep_id,
			user_id = excluded.user_id,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			score_state = excluded.score_state,
			user_calibrating = CASE WHEN ? THEN excluded.user_calibrating ELSE recovery.user_calibrating END,
			recovery_score = CASE WHEN ? THEN excluded.recovery_score ELSE recovery.recovery_score END,
			resting_heart_rate = CASE WHEN ? THEN excluded.resting_heart_rate ELSE recovery.resting_heart_rate END,
			hrv_rmssd = CASE WHEN ? THEN excluded.hrv_rmssd ELSE recovery.hrv_rmssd END,
			spo2_percentage = CASE WHEN ? THEN excluded.spo2_percentage ELSE recovery.spo2_percentage END,
			skin_temp_celsius = CASE WHEN ? THEN excluded.skin_temp_celsius ELSE recovery.skin_temp_celsius END`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range recoveries {
		var calibrating bool
		var score, rhr, hrv float64
		var spo2, skinTemp sql.NullFloat64
		hasScore := r.Score != nil
		hasSpo2, hasSkin := false, false
		if hasScore {
			calibrating = r.Score.UserCalibrating
			score = r.Score.RecoveryScore
			rhr = r.Score.RestingHeartRate
			hrv = r.Score.HRVRmssd
			if r.Score.SpO2Percentage != nil {
				hasSpo2 = true
				spo2 = sql.NullFloat64{Float64: *r.Score.SpO2Percentage, Valid: true}
			}
			if r.Score.SkinTempCelsius != nil {
				hasSkin = true
				skinTemp = sql.NullFloat64{Float64: *r.Score.SkinTempCelsius, Valid: true}
			}
		}
		hs := boolToInt(hasScore)
		if _, err := stmt.Exec(
			r.CycleID, r.SleepID, r.UserID, r.CreatedAt, r.UpdatedAt, r.ScoreState,
			calibrating, score, rhr, hrv, spo2, skinTemp,
			hs, hs, hs, hs, boolToInt(hasSpo2), boolToInt(hasSkin),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) ListRecoveries(from, to string) ([]models.Recovery, error) {
	query := `SELECT cycle_id, sleep_id, user_id, created_at, updated_at, score_state,
		user_calibrating, recovery_score, resting_heart_rate, hrv_rmssd,
		spo2_percentage, skin_temp_celsius FROM recovery WHERE 1=1`
	args := []any{}
	var err error
	query, args, err = appendListBounds(query, args, "created_at", from, to)
	if err != nil {
		return nil, err
	}
	query += ` ORDER BY created_at DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	recoveries := []models.Recovery{}
	if from != "" && to == "" {
		recoveries = make([]models.Recovery, 0, 90)
	}

	for rows.Next() {
		var r models.Recovery
		var calibrating bool
		var score, rhr, hrv float64
		var spo2, skinTemp sql.NullFloat64
		if err := rows.Scan(
			&r.CycleID, &r.SleepID, &r.UserID, &r.CreatedAt, &r.UpdatedAt, &r.ScoreState,
			&calibrating, &score, &rhr, &hrv, &spo2, &skinTemp,
		); err != nil {
			return nil, err
		}
		if r.ScoreState == "SCORED" {
			rs := &models.RecoveryScore{
				UserCalibrating:  calibrating,
				RecoveryScore:    score,
				RestingHeartRate: rhr,
				HRVRmssd:         hrv,
			}
			if spo2.Valid {
				v := spo2.Float64
				rs.SpO2Percentage = &v
			}
			if skinTemp.Valid {
				v := skinTemp.Float64
				rs.SkinTempCelsius = &v
			}
			r.Score = rs
		}
		recoveries = append(recoveries, r)
	}
	return recoveries, rows.Err()
}
