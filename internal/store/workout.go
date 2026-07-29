package store

import (
	"database/sql"

	"git.infra.hisao.org/hisao/whooper/internal/models"
)

func (db *DB) SaveWorkouts(workouts []models.Workout) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO workout
		(id, user_id, created_at, updated_at, start, end, timezone_offset, sport_id, score_state,
		 strain, average_heart_rate, max_heart_rate, kilojoule,
		 percent_recorded, distance_meter, altitude_gain_meter, altitude_change_meter,
		 zone_zero_milli, zone_one_milli, zone_two_milli,
		 zone_three_milli, zone_four_milli, zone_five_milli)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			user_id = excluded.user_id,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			start = excluded.start,
			end = excluded.end,
			timezone_offset = excluded.timezone_offset,
			sport_id = excluded.sport_id,
			score_state = excluded.score_state,
			strain = CASE WHEN ? THEN excluded.strain ELSE workout.strain END,
			average_heart_rate = CASE WHEN ? THEN excluded.average_heart_rate ELSE workout.average_heart_rate END,
			max_heart_rate = CASE WHEN ? THEN excluded.max_heart_rate ELSE workout.max_heart_rate END,
			kilojoule = CASE WHEN ? THEN excluded.kilojoule ELSE workout.kilojoule END,
			percent_recorded = CASE WHEN ? THEN excluded.percent_recorded ELSE workout.percent_recorded END,
			distance_meter = CASE WHEN ? THEN excluded.distance_meter ELSE workout.distance_meter END,
			altitude_gain_meter = CASE WHEN ? THEN excluded.altitude_gain_meter ELSE workout.altitude_gain_meter END,
			altitude_change_meter = CASE WHEN ? THEN excluded.altitude_change_meter ELSE workout.altitude_change_meter END,
			zone_zero_milli = CASE WHEN ? THEN excluded.zone_zero_milli ELSE workout.zone_zero_milli END,
			zone_one_milli = CASE WHEN ? THEN excluded.zone_one_milli ELSE workout.zone_one_milli END,
			zone_two_milli = CASE WHEN ? THEN excluded.zone_two_milli ELSE workout.zone_two_milli END,
			zone_three_milli = CASE WHEN ? THEN excluded.zone_three_milli ELSE workout.zone_three_milli END,
			zone_four_milli = CASE WHEN ? THEN excluded.zone_four_milli ELSE workout.zone_four_milli END,
			zone_five_milli = CASE WHEN ? THEN excluded.zone_five_milli ELSE workout.zone_five_milli END`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, w := range workouts {
		var strain, kj, pctRec float64
		var avgHR, maxHR int
		var dist, altGain, altChange sql.NullFloat64
		var z0, z1, z2, z3, z4, z5 int
		hasScore := w.Score != nil
		hasDist, hasAltGain, hasAltChange, hasZones := false, false, false, false
		if hasScore {
			strain = w.Score.Strain
			avgHR = w.Score.AverageHeartRate
			maxHR = w.Score.MaxHeartRate
			kj = w.Score.Kilojoule
			pctRec = w.Score.PercentRecorded
			if w.Score.DistanceMeter != nil {
				hasDist = true
				dist = sql.NullFloat64{Float64: *w.Score.DistanceMeter, Valid: true}
			}
			if w.Score.AltitudeGainMeter != nil {
				hasAltGain = true
				altGain = sql.NullFloat64{Float64: *w.Score.AltitudeGainMeter, Valid: true}
			}
			if w.Score.AltitudeChangeMeter != nil {
				hasAltChange = true
				altChange = sql.NullFloat64{Float64: *w.Score.AltitudeChangeMeter, Valid: true}
			}
			if w.Score.ZoneDuration != nil {
				hasZones = true
				z0 = w.Score.ZoneDuration.ZoneZeroMilli
				z1 = w.Score.ZoneDuration.ZoneOneMilli
				z2 = w.Score.ZoneDuration.ZoneTwoMilli
				z3 = w.Score.ZoneDuration.ZoneThreeMilli
				z4 = w.Score.ZoneDuration.ZoneFourMilli
				z5 = w.Score.ZoneDuration.ZoneFiveMilli
			}
		}
		hs := boolToInt(hasScore)
		hz := boolToInt(hasZones)
		if _, err := stmt.Exec(
			w.ID, w.UserID, w.CreatedAt, w.UpdatedAt, w.Start, w.End, w.TimezoneOffset, w.SportID, w.ScoreState,
			strain, avgHR, maxHR, kj,
			pctRec, dist, altGain, altChange,
			z0, z1, z2, z3, z4, z5,
			hs, hs, hs, hs, hs,
			boolToInt(hasDist), boolToInt(hasAltGain), boolToInt(hasAltChange),
			hz, hz, hz, hz, hz, hz,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) ListWorkouts(from, to string) ([]models.Workout, error) {
	query := `SELECT id, user_id, created_at, updated_at, start, end, timezone_offset, sport_id, score_state,
		strain, average_heart_rate, max_heart_rate, kilojoule,
		percent_recorded, distance_meter, altitude_gain_meter, altitude_change_meter,
		zone_zero_milli, zone_one_milli, zone_two_milli,
		zone_three_milli, zone_four_milli, zone_five_milli
		FROM workout WHERE 1=1`
	args := []any{}
	var err error
	query, args, err = appendListBounds(query, args, "start", from, to)
	if err != nil {
		return nil, err
	}
	query += ` ORDER BY start DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workouts := []models.Workout{}
	if from != "" && to == "" {
		workouts = make([]models.Workout, 0, 90)
	}

	for rows.Next() {
		var w models.Workout
		var strain, kj, pctRec float64
		var avgHR, maxHR int
		var dist, altGain, altChange sql.NullFloat64
		var z0, z1, z2, z3, z4, z5 int
		if err := rows.Scan(
			&w.ID, &w.UserID, &w.CreatedAt, &w.UpdatedAt, &w.Start, &w.End, &w.TimezoneOffset, &w.SportID, &w.ScoreState,
			&strain, &avgHR, &maxHR, &kj,
			&pctRec, &dist, &altGain, &altChange,
			&z0, &z1, &z2, &z3, &z4, &z5,
		); err != nil {
			return nil, err
		}
		if w.ScoreState == "SCORED" {
			ws := &models.WorkoutScore{
				Strain:           strain,
				AverageHeartRate: avgHR,
				MaxHeartRate:     maxHR,
				Kilojoule:        kj,
				PercentRecorded:  pctRec,
				ZoneDuration: &models.ZoneDuration{
					ZoneZeroMilli:  z0,
					ZoneOneMilli:   z1,
					ZoneTwoMilli:   z2,
					ZoneThreeMilli: z3,
					ZoneFourMilli:  z4,
					ZoneFiveMilli:  z5,
				},
			}
			if dist.Valid {
				v := dist.Float64
				ws.DistanceMeter = &v
			}
			if altGain.Valid {
				v := altGain.Float64
				ws.AltitudeGainMeter = &v
			}
			if altChange.Valid {
				v := altChange.Float64
				ws.AltitudeChangeMeter = &v
			}
			w.Score = ws
		}
		workouts = append(workouts, w)
	}
	return workouts, rows.Err()
}
