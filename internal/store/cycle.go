package store

import (
	"git.infra.hisao.org/hisao/whooper/internal/models"
)

func (db *DB) SaveCycles(cycles []models.Cycle) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO cycle
		(id, user_id, created_at, updated_at, start, end, timezone_offset, days, score_state,
		 strain, kilojoule, average_heart_rate, max_heart_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			user_id = excluded.user_id,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			start = excluded.start,
			end = excluded.end,
			timezone_offset = excluded.timezone_offset,
			days = excluded.days,
			score_state = excluded.score_state,
			strain = CASE WHEN ? THEN excluded.strain ELSE cycle.strain END,
			kilojoule = CASE WHEN ? THEN excluded.kilojoule ELSE cycle.kilojoule END,
			average_heart_rate = CASE WHEN ? THEN excluded.average_heart_rate ELSE cycle.average_heart_rate END,
			max_heart_rate = CASE WHEN ? THEN excluded.max_heart_rate ELSE cycle.max_heart_rate END`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range cycles {
		var strain, kj float64
		var avgHR, maxHR int
		hasScore := c.Score != nil
		if hasScore {
			strain = c.Score.Strain
			kj = c.Score.Kilojoule
			avgHR = c.Score.AverageHeartRate
			maxHR = c.Score.MaxHeartRate
		}
		hs := boolToInt(hasScore)
		if _, err := stmt.Exec(
			c.ID, c.UserID, c.CreatedAt, c.UpdatedAt, c.Start, c.End, c.TimezoneOffset, c.Days, c.ScoreState,
			strain, kj, avgHR, maxHR,
			hs, hs, hs, hs,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) ListCycles(from, to string) ([]models.Cycle, error) {
	query := `SELECT id, user_id, created_at, updated_at, start, end, timezone_offset, days, score_state,
		strain, kilojoule, average_heart_rate, max_heart_rate FROM cycle WHERE 1=1`
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

	cycles := []models.Cycle{}
	if from != "" && to == "" {
		cycles = make([]models.Cycle, 0, 90)
	}

	for rows.Next() {
		var c models.Cycle
		var strain, kj float64
		var avgHR, maxHR int
		if err := rows.Scan(
			&c.ID, &c.UserID, &c.CreatedAt, &c.UpdatedAt, &c.Start, &c.End, &c.TimezoneOffset, &c.Days, &c.ScoreState,
			&strain, &kj, &avgHR, &maxHR,
		); err != nil {
			return nil, err
		}
		if c.ScoreState == "SCORED" {
			c.Score = &models.CycleScore{
				Strain:           strain,
				Kilojoule:        kj,
				AverageHeartRate: avgHR,
				MaxHeartRate:     maxHR,
			}
		}
		cycles = append(cycles, c)
	}
	return cycles, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
