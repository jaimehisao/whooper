package store

import (
	"database/sql"
	"fmt"
)

// RecoveryTrendPoint holds a single day's recovery trend data.
type RecoveryTrendPoint struct {
	Date          string
	RecoveryScore float64
	HRV           float64
	RHR           float64
}

// SleepTrendPoint holds a single day's sleep trend data.
type SleepTrendPoint struct {
	Date               string
	DurationMilli      int
	EfficiencyPct      float64
	PerformancePct     float64
	ConsistencyPct     float64
}

// StrainTrendPoint holds a single day's strain trend data.
type StrainTrendPoint struct {
	Date          string
	Strain        float64
	MaxHeartRate  int
	Kilojoule     float64
}

// CorrelationPoint holds a pair of metric values for correlation analysis.
type CorrelationPoint struct {
	X float64
	Y float64
}

func (db *DB) GetRecoveryTrend(from, to string) ([]RecoveryTrendPoint, error) {
	query := `SELECT date(created_at) AS d, recovery_score, hrv_rmssd, resting_heart_rate
		FROM recovery WHERE score_state = 'SCORED'`
	args := []any{}

	if from != "" {
		query += ` AND created_at >= ?`
		args = append(args, from)
	}
	if to != "" {
		query += ` AND created_at <= ?`
		args = append(args, to)
	}
	query += ` ORDER BY d`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []RecoveryTrendPoint
	for rows.Next() {
		var p RecoveryTrendPoint
		if err := rows.Scan(&p.Date, &p.RecoveryScore, &p.HRV, &p.RHR); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func (db *DB) GetSleepTrend(from, to string) ([]SleepTrendPoint, error) {
	query := `SELECT date(start) AS d,
		total_in_bed_time_milli - total_awake_time_milli AS duration_milli,
		sleep_efficiency_pct, sleep_performance_pct, sleep_consistency_pct
		FROM sleep WHERE nap = 0 AND score_state = 'SCORED'`
	args := []any{}

	if from != "" {
		query += ` AND start >= ?`
		args = append(args, from)
	}
	if to != "" {
		query += ` AND start <= ?`
		args = append(args, to)
	}
	query += ` ORDER BY d`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []SleepTrendPoint
	for rows.Next() {
		var p SleepTrendPoint
		if err := rows.Scan(&p.Date, &p.DurationMilli, &p.EfficiencyPct, &p.PerformancePct, &p.ConsistencyPct); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func (db *DB) GetStrainTrend(from, to string) ([]StrainTrendPoint, error) {
	query := `SELECT date(start) AS d, strain, max_heart_rate, kilojoule
		FROM cycle WHERE score_state = 'SCORED'`
	args := []any{}

	if from != "" {
		query += ` AND start >= ?`
		args = append(args, from)
	}
	if to != "" {
		query += ` AND start <= ?`
		args = append(args, to)
	}
	query += ` ORDER BY d`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []StrainTrendPoint
	for rows.Next() {
		var p StrainTrendPoint
		if err := rows.Scan(&p.Date, &p.Strain, &p.MaxHeartRate, &p.Kilojoule); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func (db *DB) GetCorrelationData(metricX, metricY string) ([]CorrelationPoint, error) {
	colX, tableX, err := metricColumn(metricX)
	if err != nil {
		return nil, err
	}
	colY, tableY, err := metricColumn(metricY)
	if err != nil {
		return nil, err
	}

	// Join through cycle to align data by date.
	query := fmt.Sprintf(`SELECT a.%s, b.%s FROM`, colX, colY)

	if tableX == tableY {
		query = fmt.Sprintf(`SELECT %s, %s FROM %s WHERE score_state = 'SCORED'`,
			colX, colY, tableX)
	} else {
		query = fmt.Sprintf(
			`SELECT a.%s, b.%s
			FROM %s a
			JOIN %s b ON date(a.%s) = date(b.%s)
			WHERE a.score_state = 'SCORED' AND b.score_state = 'SCORED'`,
			colX, colY,
			tableX, tableY,
			dateColumn(tableX), dateColumn(tableY),
		)
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []CorrelationPoint
	for rows.Next() {
		var p CorrelationPoint
		if err := rows.Scan(&p.X, &p.Y); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func metricColumn(metric string) (column, table string, err error) {
	switch metric {
	case "recovery":
		return "recovery_score", "recovery", nil
	case "hrv":
		return "hrv_rmssd", "recovery", nil
	case "rhr":
		return "resting_heart_rate", "recovery", nil
	case "strain":
		return "strain", "cycle", nil
	case "sleep_duration":
		return "(total_in_bed_time_milli - total_awake_time_milli)", "sleep", nil
	case "sleep_efficiency":
		return "sleep_efficiency_pct", "sleep", nil
	default:
		return "", "", fmt.Errorf("unknown metric: %s", metric)
	}
}

func dateColumn(table string) string {
	switch table {
	case "recovery":
		return "created_at"
	default:
		return "start"
	}
}

func (db *DB) GetSyncState(entity string) (string, error) {
	var lastSynced string
	err := db.QueryRow(`SELECT last_synced FROM sync_state WHERE entity = ?`, entity).
		Scan(&lastSynced)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return lastSynced, err
}

func (db *DB) SetSyncState(entity, lastSynced string) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO sync_state (entity, last_synced) VALUES (?, ?)`,
		entity, lastSynced)
	return err
}
