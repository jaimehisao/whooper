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

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO cycle
		(id, user_id, created_at, updated_at, start, end, days, score_state,
		 strain, kilojoule, average_heart_rate, max_heart_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range cycles {
		var strain, kj float64
		var avgHR, maxHR int
		if c.Score != nil {
			strain = c.Score.Strain
			kj = c.Score.Kilojoule
			avgHR = c.Score.AverageHeartRate
			maxHR = c.Score.MaxHeartRate
		}
		if _, err := stmt.Exec(
			c.ID, c.UserID, c.CreatedAt, c.UpdatedAt, c.Start, c.End, c.Days, c.ScoreState,
			strain, kj, avgHR, maxHR,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) ListCycles(from, to string) ([]models.Cycle, error) {
	query := `SELECT id, user_id, created_at, updated_at, start, end, days, score_state,
		strain, kilojoule, average_heart_rate, max_heart_rate FROM cycle WHERE 1=1`
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

	cycles := []models.Cycle{}
	if from != "" && to == "" {
		cycles = make([]models.Cycle, 0, 90)
	}

	for rows.Next() {
		var c models.Cycle
		var strain, kj float64
		var avgHR, maxHR int
		if err := rows.Scan(
			&c.ID, &c.UserID, &c.CreatedAt, &c.UpdatedAt, &c.Start, &c.End, &c.Days, &c.ScoreState,
			&strain, &kj, &avgHR, &maxHR,
		); err != nil {
			return nil, err
		}
		c.Score = &models.CycleScore{
			Strain:           strain,
			Kilojoule:        kj,
			AverageHeartRate: avgHR,
			MaxHeartRate:     maxHR,
		}
		cycles = append(cycles, c)
	}
	return cycles, rows.Err()
}
