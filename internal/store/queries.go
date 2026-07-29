package store

import (
	"database/sql"
	"fmt"
)

// DefaultCorrelationLookbackDays caps correlation queries to recent history.
const DefaultCorrelationLookbackDays = 365

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
	fromBound, toBound, err := ExpandRangeBounds(from, to)
	if err != nil {
		return nil, err
	}

	dayExpr := `date(created_at)`
	query := `SELECT ` + dayExpr + ` AS d,
		AVG(recovery_score), AVG(hrv_rmssd), AVG(resting_heart_rate)
		FROM recovery WHERE score_state = 'SCORED'`
	args := []any{}

	if fromBound != "" {
		query += ` AND created_at >= ?`
		args = append(args, fromBound)
	}
	if toBound != "" {
		query += ` AND created_at <= ?`
		args = append(args, toBound)
	}
	query += ` GROUP BY d ORDER BY d ASC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []RecoveryTrendPoint
	if fromBound != "" && toBound == "" {
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
	fromBound, toBound, err := ExpandRangeBounds(from, to)
	if err != nil {
		return nil, err
	}

	dayExpr := localDateSQL("start")
	query := `SELECT ` + dayExpr + ` AS d,
		AVG(total_in_bed_time_milli - total_awake_time_milli),
		AVG(sleep_efficiency_pct), AVG(sleep_performance_pct), AVG(sleep_consistency_pct)
		FROM sleep WHERE nap = 0 AND score_state = 'SCORED'`
	args := []any{}

	if fromBound != "" {
		query += ` AND start >= ?`
		args = append(args, fromBound)
	}
	if toBound != "" {
		query += ` AND start <= ?`
		args = append(args, toBound)
	}
	query += ` GROUP BY d ORDER BY d ASC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []SleepTrendPoint
	if fromBound != "" && toBound == "" {
		points = make([]SleepTrendPoint, 0, 90)
	}

	for rows.Next() {
		var p SleepTrendPoint
		var duration float64
		if err := rows.Scan(&p.Date, &duration, &p.EfficiencyPct, &p.PerformancePct, &p.ConsistencyPct); err != nil {
			return nil, err
		}
		p.DurationMilli = int(duration)
		points = append(points, p)
	}
	return points, rows.Err()
}

func (db *DB) GetStrainTrend(from, to string) ([]StrainTrendPoint, error) {
	fromBound, toBound, err := ExpandRangeBounds(from, to)
	if err != nil {
		return nil, err
	}

	dayExpr := localDateSQL("start")
	query := `SELECT ` + dayExpr + ` AS d,
		AVG(strain), AVG(max_heart_rate), AVG(kilojoule)
		FROM cycle WHERE score_state = 'SCORED'`
	args := []any{}

	if fromBound != "" {
		query += ` AND start >= ?`
		args = append(args, fromBound)
	}
	if toBound != "" {
		query += ` AND start <= ?`
		args = append(args, toBound)
	}
	query += ` GROUP BY d ORDER BY d ASC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []StrainTrendPoint
	if fromBound != "" && toBound == "" {
		points = make([]StrainTrendPoint, 0, 90)
	}

	for rows.Next() {
		var p StrainTrendPoint
		var maxHR float64
		if err := rows.Scan(&p.Date, &p.Strain, &maxHR, &p.Kilojoule); err != nil {
			return nil, err
		}
		p.MaxHeartRate = int(maxHR)
		points = append(points, p)
	}
	return points, rows.Err()
}

func (db *DB) GetCorrelationData(metricX, metricY string) ([]CorrelationPoint, error) {
	return db.GetCorrelationDataSince(metricX, metricY, DefaultCorrelationLookbackDays)
}

func (db *DB) GetCorrelationDataSince(metricX, metricY string, sinceDays int) ([]CorrelationPoint, error) {
	if sinceDays < 0 {
		sinceDays = DefaultCorrelationLookbackDays
	}

	colX, tableX, err := metricColumn(metricX)
	if err != nil {
		return nil, err
	}
	colY, tableY, err := metricColumn(metricY)
	if err != nil {
		return nil, err
	}

	napX := scoredNapFilter(tableX)
	napY := scoredNapFilter(tableY)
	lookbackX, lookbackY := "1=1", "1=1"
	if sinceDays > 0 {
		lookbackX = fmt.Sprintf(`date(%s) >= date('now', '-%d days')`, dateColumn(tableX), sinceDays)
		lookbackY = fmt.Sprintf(`date(%s) >= date('now', '-%d days')`, dateColumn(tableY), sinceDays)
	}

	var query string
	if tableX == tableY {
		dayExpr := correlationDayExpr(tableX)
		query = fmt.Sprintf(
			`SELECT x, y FROM (
				SELECT %s AS d, AVG(%s) AS x, AVG(%s) AS y
				FROM %s
				WHERE score_state = 'SCORED'%s AND %s
				GROUP BY d
			)`,
			dayExpr, colX, colY, tableX, napX, lookbackX,
		)
	} else {
		dayX := correlationDayExpr(tableX)
		dayY := correlationDayExpr(tableY)
		query = fmt.Sprintf(
			`WITH a_daily AS (
				SELECT %s AS d, AVG(%s) AS x
				FROM %s
				WHERE score_state = 'SCORED'%s AND %s
				GROUP BY d
			),
			b_daily AS (
				SELECT %s AS d, AVG(%s) AS y
				FROM %s
				WHERE score_state = 'SCORED'%s AND %s
				GROUP BY d
			)
			SELECT a_daily.x, b_daily.y
			FROM a_daily
			JOIN b_daily ON a_daily.d = b_daily.d`,
			dayX, colX, tableX, napX, lookbackX,
			dayY, colY, tableY, napY, lookbackY,
		)
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	capHint := DefaultCorrelationLookbackDays
	if sinceDays > 0 {
		capHint = sinceDays
	}
	points := make([]CorrelationPoint, 0, capHint)
	for rows.Next() {
		var p CorrelationPoint
		if err := rows.Scan(&p.X, &p.Y); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func scoredNapFilter(table string) string {
	if table == "sleep" {
		return ` AND nap = 0`
	}
	return ""
}

func correlationDayExpr(table string) string {
	col := dateColumn(table)
	if table == "recovery" {
		return `date(created_at)`
	}
	return localDateSQL(col)
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

// EarliestPendingActivityStart returns the earliest activity start (or recovery created_at)
// among rows still in PENDING_SCORE state. Empty string means none pending.
func (db *DB) EarliestPendingActivityStart() (string, error) {
	var earliest sql.NullString
	err := db.QueryRow(`
		SELECT MIN(ts) FROM (
			SELECT MIN(start) AS ts FROM cycle WHERE score_state = 'PENDING_SCORE'
			UNION ALL
			SELECT MIN(start) FROM sleep WHERE score_state = 'PENDING_SCORE'
			UNION ALL
			SELECT MIN(start) FROM workout WHERE score_state = 'PENDING_SCORE'
			UNION ALL
			SELECT MIN(created_at) FROM recovery WHERE score_state = 'PENDING_SCORE'
		)`).Scan(&earliest)
	if err != nil {
		return "", err
	}
	if !earliest.Valid {
		return "", nil
	}
	return earliest.String, nil
}

