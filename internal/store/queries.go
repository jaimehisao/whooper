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
	Date           string
	DurationMilli  int
	EfficiencyPct  float64
	PerformancePct float64
	ConsistencyPct float64
}

// StrainTrendPoint holds a single day's strain trend data.
type StrainTrendPoint struct {
	Date         string
	Strain       float64
	MaxHeartRate int
	Kilojoule    float64
}

// CorrelationPoint holds a pair of metric values for correlation analysis.
type CorrelationPoint struct {
	X float64
	Y float64
}

func (db *DB) GetRecoveryTrend(from, to string) ([]RecoveryTrendPoint, error) {
	from, to = NormalizeBounds(from, to)
	// One row per calendar day: pick the latest scored recovery that day.
	query := `SELECT date(r.created_at) AS d, r.recovery_score, r.hrv_rmssd, r.resting_heart_rate
		FROM recovery r
		INNER JOIN (
			SELECT date(created_at) AS d, MAX(created_at) AS max_created
			FROM recovery
			WHERE score_state = 'SCORED'`
	args := []any{}
	if from != "" {
		query += ` AND created_at >= ?`
		args = append(args, from)
	}
	if to != "" {
		query += ` AND created_at <= ?`
		args = append(args, to)
	}
	query += `
			GROUP BY date(created_at)
		) latest ON date(r.created_at) = latest.d AND r.created_at = latest.max_created
		WHERE r.score_state = 'SCORED'`
	if from != "" {
		query += ` AND r.created_at >= ?`
		args = append(args, from)
	}
	if to != "" {
		query += ` AND r.created_at <= ?`
		args = append(args, to)
	}
	query += ` ORDER BY d`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []RecoveryTrendPoint
	if from != "" && to == "" {
		points = make([]RecoveryTrendPoint, 0, 90)
	}

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
	from, to = NormalizeBounds(from, to)
	query := `SELECT date(s.start) AS d,
		s.total_in_bed_time_milli - s.total_awake_time_milli AS duration_milli,
		s.sleep_efficiency_pct, s.sleep_performance_pct, s.sleep_consistency_pct
		FROM sleep s
		INNER JOIN (
			SELECT date(start) AS d, MAX(start) AS max_start
			FROM sleep
			WHERE nap = 0 AND score_state = 'SCORED'`
	args := []any{}
	if from != "" {
		query += ` AND start >= ?`
		args = append(args, from)
	}
	if to != "" {
		query += ` AND start <= ?`
		args = append(args, to)
	}
	query += `
			GROUP BY date(start)
		) latest ON date(s.start) = latest.d AND s.start = latest.max_start
		WHERE s.nap = 0 AND s.score_state = 'SCORED'`
	if from != "" {
		query += ` AND s.start >= ?`
		args = append(args, from)
	}
	if to != "" {
		query += ` AND s.start <= ?`
		args = append(args, to)
	}
	query += ` ORDER BY d`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []SleepTrendPoint
	if from != "" && to == "" {
		points = make([]SleepTrendPoint, 0, 90)
	}

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
	from, to = NormalizeBounds(from, to)
	query := `SELECT date(c.start) AS d, c.strain, c.max_heart_rate, c.kilojoule
		FROM cycle c
		INNER JOIN (
			SELECT date(start) AS d, MAX(start) AS max_start
			FROM cycle
			WHERE score_state = 'SCORED'`
	args := []any{}
	if from != "" {
		query += ` AND start >= ?`
		args = append(args, from)
	}
	if to != "" {
		query += ` AND start <= ?`
		args = append(args, to)
	}
	query += `
			GROUP BY date(start)
		) latest ON date(c.start) = latest.d AND c.start = latest.max_start
		WHERE c.score_state = 'SCORED'`
	if from != "" {
		query += ` AND c.start >= ?`
		args = append(args, from)
	}
	if to != "" {
		query += ` AND c.start <= ?`
		args = append(args, to)
	}
	query += ` ORDER BY d`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []StrainTrendPoint
	if from != "" && to == "" {
		points = make([]StrainTrendPoint, 0, 90)
	}

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
	return db.GetCorrelationDataInRange(metricX, metricY, "", "")
}

// GetCorrelationDataInRange returns correlation points optionally limited to [from, to].
// Empty bounds load all scored history (callers should prefer a bounded window).
func (db *DB) GetCorrelationDataInRange(metricX, metricY, from, to string) ([]CorrelationPoint, error) {
	from, to = NormalizeBounds(from, to)
	colX, tableX, err := metricColumn(metricX)
	if err != nil {
		return nil, err
	}
	colY, tableY, err := metricColumn(metricY)
	if err != nil {
		return nil, err
	}

	var query string
	args := []any{}
	if tableX == tableY {
		dateCol := dateColumn(tableX)
		query = fmt.Sprintf(
			`SELECT AVG(%s), AVG(%s) FROM %s WHERE score_state = 'SCORED'`,
			colX, colY, tableX,
		)
		if from != "" {
			query += fmt.Sprintf(` AND %s >= ?`, dateCol)
			args = append(args, from)
		}
		if to != "" {
			query += fmt.Sprintf(` AND %s <= ?`, dateCol)
			args = append(args, to)
		}
		query += fmt.Sprintf(` GROUP BY date(%s) ORDER BY date(%s)`, dateCol, dateCol)
	} else {
		dateX := dateColumn(tableX)
		dateY := dateColumn(tableY)
		aFilter := "WHERE score_state = 'SCORED'"
		bFilter := "WHERE score_state = 'SCORED'"
		aArgs := []any{}
		bArgs := []any{}
		if from != "" {
			aFilter += fmt.Sprintf(` AND %s >= ?`, dateX)
			bFilter += fmt.Sprintf(` AND %s >= ?`, dateY)
			aArgs = append(aArgs, from)
			bArgs = append(bArgs, from)
		}
		if to != "" {
			aFilter += fmt.Sprintf(` AND %s <= ?`, dateX)
			bFilter += fmt.Sprintf(` AND %s <= ?`, dateY)
			aArgs = append(aArgs, to)
			bArgs = append(bArgs, to)
		}
		query = fmt.Sprintf(
			`WITH a_daily AS (
				SELECT date(%s) AS d, AVG(%s) AS x
				FROM %s
				%s
				GROUP BY date(%s)
			),
			b_daily AS (
				SELECT date(%s) AS d, AVG(%s) AS y
				FROM %s
				%s
				GROUP BY date(%s)
			)
			SELECT a_daily.x, b_daily.y
			FROM a_daily
			JOIN b_daily ON a_daily.d = b_daily.d`,
			dateX, colX, tableX, aFilter, dateX,
			dateY, colY, tableY, bFilter, dateY,
		)
		args = append(args, aArgs...)
		args = append(args, bArgs...)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]CorrelationPoint, 0, 365)
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
