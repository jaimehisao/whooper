package store

import (
	"git.infra.hisao.org/hisao/whooper/internal/models"
)

func (db *DB) SaveRecoveries(recoveries []models.Recovery) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO recovery
		(cycle_id, sleep_id, user_id, created_at, updated_at, score_state,
		 user_calibrating, recovery_score, resting_heart_rate, hrv_rmssd,
		 spo2_percentage, skin_temp_celsius)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range recoveries {
		var calibrating bool
		var score, rhr, hrv, spo2, skinTemp float64
		if r.Score != nil {
			calibrating = r.Score.UserCalibrating
			score = r.Score.RecoveryScore
			rhr = r.Score.RestingHeartRate
			hrv = r.Score.HRVRmssd
			spo2 = r.Score.SpO2Percentage
			skinTemp = r.Score.SkinTempCelsius
		}
		if _, err := stmt.Exec(
			r.CycleID, r.SleepID, r.UserID, r.CreatedAt, r.UpdatedAt, r.ScoreState,
			calibrating, score, rhr, hrv, spo2, skinTemp,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) ListRecoveries(from, to string) ([]models.Recovery, error) {
	from, to = NormalizeBounds(from, to)
	query := `SELECT cycle_id, sleep_id, user_id, created_at, updated_at, score_state,
		user_calibrating, recovery_score, resting_heart_rate, hrv_rmssd,
		spo2_percentage, skin_temp_celsius FROM recovery WHERE 1=1`
	args := []any{}

	if from != "" {
		query += ` AND created_at >= ?`
		args = append(args, from)
	}
	if to != "" {
		query += ` AND created_at <= ?`
		args = append(args, to)
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
		var score, rhr, hrv, spo2, skinTemp float64
		if err := rows.Scan(
			&r.CycleID, &r.SleepID, &r.UserID, &r.CreatedAt, &r.UpdatedAt, &r.ScoreState,
			&calibrating, &score, &rhr, &hrv, &spo2, &skinTemp,
		); err != nil {
			return nil, err
		}
		r.Score = &models.RecoveryScore{
			UserCalibrating:  calibrating,
			RecoveryScore:    score,
			RestingHeartRate: rhr,
			HRVRmssd:         hrv,
			SpO2Percentage:   spo2,
			SkinTempCelsius:  skinTemp,
		}
		recoveries = append(recoveries, r)
	}
	return recoveries, rows.Err()
}
