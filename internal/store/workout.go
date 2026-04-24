package store

import (
	"git.infra.hisao.org/hisao/whooper/internal/models"
)

func (db *DB) SaveWorkouts(workouts []models.Workout) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO workout
		(id, user_id, created_at, updated_at, start, end, sport_id, score_state,
		 strain, average_heart_rate, max_heart_rate, kilojoule,
		 percent_recorded, distance_meter, altitude_gain_meter, altitude_change_meter,
		 zone_zero_milli, zone_one_milli, zone_two_milli,
		 zone_three_milli, zone_four_milli, zone_five_milli)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, w := range workouts {
		var strain, kj, pctRec, dist, altGain, altChange float64
		var avgHR, maxHR int
		var z0, z1, z2, z3, z4, z5 int
		if w.Score != nil {
			strain = w.Score.Strain
			avgHR = w.Score.AverageHeartRate
			maxHR = w.Score.MaxHeartRate
			kj = w.Score.Kilojoule
			pctRec = w.Score.PercentRecorded
			dist = w.Score.DistanceMeter
			altGain = w.Score.AltitudeGainMeter
			altChange = w.Score.AltitudeChangeMeter
			if w.Score.ZoneDuration != nil {
				z0 = w.Score.ZoneDuration.ZoneZeroMilli
				z1 = w.Score.ZoneDuration.ZoneOneMilli
				z2 = w.Score.ZoneDuration.ZoneTwoMilli
				z3 = w.Score.ZoneDuration.ZoneThreeMilli
				z4 = w.Score.ZoneDuration.ZoneFourMilli
				z5 = w.Score.ZoneDuration.ZoneFiveMilli
			}
		}
		if _, err := stmt.Exec(
			w.ID, w.UserID, w.CreatedAt, w.UpdatedAt, w.Start, w.End, w.SportID, w.ScoreState,
			strain, avgHR, maxHR, kj,
			pctRec, dist, altGain, altChange,
			z0, z1, z2, z3, z4, z5,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) ListWorkouts(from, to string) ([]models.Workout, error) {
	query := `SELECT id, user_id, created_at, updated_at, start, end, sport_id, score_state,
		strain, average_heart_rate, max_heart_rate, kilojoule,
		percent_recorded, distance_meter, altitude_gain_meter, altitude_change_meter,
		zone_zero_milli, zone_one_milli, zone_two_milli,
		zone_three_milli, zone_four_milli, zone_five_milli
		FROM workout WHERE 1=1`
	args := []any{}

	if from != "" {
		query += ` AND start >= ?`
		args = append(args, from)
	}
	if to != "" {
		query += ` AND start <= ?`
		args = append(args, to)
	}
	query += ` ORDER BY start DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workouts []models.Workout
	if from != "" && to == "" {
		workouts = make([]models.Workout, 0, 90)
	}

	for rows.Next() {
		var w models.Workout
		var strain, kj, pctRec, dist, altGain, altChange float64
		var avgHR, maxHR int
		var z0, z1, z2, z3, z4, z5 int
		if err := rows.Scan(
			&w.ID, &w.UserID, &w.CreatedAt, &w.UpdatedAt, &w.Start, &w.End, &w.SportID, &w.ScoreState,
			&strain, &avgHR, &maxHR, &kj,
			&pctRec, &dist, &altGain, &altChange,
			&z0, &z1, &z2, &z3, &z4, &z5,
		); err != nil {
			return nil, err
		}
		w.Score = &models.WorkoutScore{
			Strain:              strain,
			AverageHeartRate:    avgHR,
			MaxHeartRate:        maxHR,
			Kilojoule:           kj,
			PercentRecorded:     pctRec,
			DistanceMeter:       dist,
			AltitudeGainMeter:   altGain,
			AltitudeChangeMeter: altChange,
			ZoneDuration: &models.ZoneDuration{
				ZoneZeroMilli:  z0,
				ZoneOneMilli:   z1,
				ZoneTwoMilli:   z2,
				ZoneThreeMilli: z3,
				ZoneFourMilli:  z4,
				ZoneFiveMilli:  z5,
			},
		}
		workouts = append(workouts, w)
	}
	return workouts, rows.Err()
}
