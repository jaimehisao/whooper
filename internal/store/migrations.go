package store

import (
	"database/sql"
	"fmt"
)

// migrationVersion tracks which migrations have been applied.
// Each migration has a version number and a list of SQL statements.
type migration struct {
	version    int
	statements []string
}

var migrations = []migration{
	{
		version: 1,
		statements: []string{
			`CREATE TABLE IF NOT EXISTS profile (
				user_id    INTEGER PRIMARY KEY,
				email      TEXT NOT NULL,
				first_name TEXT NOT NULL,
				last_name  TEXT NOT NULL
			)`,
			`CREATE TABLE IF NOT EXISTS cycle (
				id                 INTEGER PRIMARY KEY,
				user_id            INTEGER NOT NULL,
				created_at         TEXT NOT NULL,
				updated_at         TEXT NOT NULL,
				start              TEXT NOT NULL,
				end                TEXT,
				days               INTEGER,
				score_state        TEXT,
				strain             REAL,
				kilojoule          REAL,
				average_heart_rate INTEGER,
				max_heart_rate     INTEGER
			)`,
			`CREATE TABLE IF NOT EXISTS recovery (
				cycle_id           INTEGER PRIMARY KEY,
				sleep_id           INTEGER,
				user_id            INTEGER NOT NULL,
				created_at         TEXT NOT NULL,
				updated_at         TEXT NOT NULL,
				score_state        TEXT,
				user_calibrating   INTEGER,
				recovery_score     REAL,
				resting_heart_rate REAL,
				hrv_rmssd          REAL,
				spo2_percentage    REAL,
				skin_temp_celsius  REAL
			)`,
			`CREATE TABLE IF NOT EXISTS sleep (
				id                            INTEGER PRIMARY KEY,
				user_id                       INTEGER NOT NULL,
				created_at                    TEXT NOT NULL,
				updated_at                    TEXT NOT NULL,
				start                         TEXT NOT NULL,
				end                           TEXT,
				nap                           INTEGER NOT NULL DEFAULT 0,
				score_state                   TEXT,
				total_in_bed_time_milli       INTEGER,
				total_awake_time_milli        INTEGER,
				total_no_data_time_milli      INTEGER,
				total_light_sleep_time_milli  INTEGER,
				total_slow_wave_sleep_time_milli INTEGER,
				total_rem_sleep_time_milli    INTEGER,
				sleep_cycle_count             INTEGER,
				disturbance_count             INTEGER,
				baseline_sleep_needed_milli   INTEGER,
				need_from_sleep_debt_milli    INTEGER,
				need_from_recent_strain_milli INTEGER,
				need_from_recent_nap_milli    INTEGER,
				respiratory_rate              REAL,
				sleep_performance_pct         REAL,
				sleep_consistency_pct         REAL,
				sleep_efficiency_pct          REAL
			)`,
			`CREATE TABLE IF NOT EXISTS workout (
				id                    INTEGER PRIMARY KEY,
				user_id               INTEGER NOT NULL,
				created_at            TEXT NOT NULL,
				updated_at            TEXT NOT NULL,
				start                 TEXT NOT NULL,
				end                   TEXT,
				sport_id              INTEGER,
				score_state           TEXT,
				strain                REAL,
				average_heart_rate    INTEGER,
				max_heart_rate        INTEGER,
				kilojoule             REAL,
				percent_recorded      REAL,
				distance_meter        REAL,
				altitude_gain_meter   REAL,
				altitude_change_meter REAL,
				zone_zero_milli       INTEGER,
				zone_one_milli        INTEGER,
				zone_two_milli        INTEGER,
				zone_three_milli      INTEGER,
				zone_four_milli       INTEGER,
				zone_five_milli       INTEGER
			)`,
			`CREATE TABLE IF NOT EXISTS sync_state (
				entity     TEXT PRIMARY KEY,
				last_synced TEXT NOT NULL
			)`,
			`CREATE INDEX IF NOT EXISTS idx_cycle_start ON cycle(start)`,
			`CREATE INDEX IF NOT EXISTS idx_recovery_created ON recovery(created_at)`,
			`CREATE INDEX IF NOT EXISTS idx_sleep_start ON sleep(start)`,
			`CREATE INDEX IF NOT EXISTS idx_workout_start ON workout(start)`,
		},
	},
	{
		version: 2,
		statements: []string{
			// Composite indexes optimized for common query patterns
			`CREATE INDEX IF NOT EXISTS idx_recovery_scored_date ON recovery(score_state, created_at)`,
			`CREATE INDEX IF NOT EXISTS idx_cycle_scored_start ON cycle(score_state, start)`,
			`CREATE INDEX IF NOT EXISTS idx_sleep_scored ON sleep(nap, score_state, start)`,
			`CREATE INDEX IF NOT EXISTS idx_workout_scored_start ON workout(score_state, start)`,
		},
	},
	{
		version: 3,
		statements: []string{
			`ALTER TABLE recovery RENAME TO recovery_old`,
			`CREATE TABLE recovery (
				cycle_id           INTEGER PRIMARY KEY,
				sleep_id           TEXT,
				user_id            INTEGER NOT NULL,
				created_at         TEXT NOT NULL,
				updated_at         TEXT NOT NULL,
				score_state        TEXT,
				user_calibrating   INTEGER,
				recovery_score     REAL,
				resting_heart_rate REAL,
				hrv_rmssd          REAL,
				spo2_percentage    REAL,
				skin_temp_celsius  REAL
			)`,
			`INSERT INTO recovery (
				cycle_id, sleep_id, user_id, created_at, updated_at, score_state,
				user_calibrating, recovery_score, resting_heart_rate, hrv_rmssd,
				spo2_percentage, skin_temp_celsius
			)
			SELECT cycle_id, CAST(sleep_id AS TEXT), user_id, created_at, updated_at, score_state,
				user_calibrating, recovery_score, resting_heart_rate, hrv_rmssd,
				spo2_percentage, skin_temp_celsius
			FROM recovery_old`,
			`DROP TABLE recovery_old`,

			`ALTER TABLE sleep RENAME TO sleep_old`,
			`CREATE TABLE sleep (
				id                            TEXT PRIMARY KEY,
				user_id                       INTEGER NOT NULL,
				created_at                    TEXT NOT NULL,
				updated_at                    TEXT NOT NULL,
				start                         TEXT NOT NULL,
				end                           TEXT,
				nap                           INTEGER NOT NULL DEFAULT 0,
				score_state                   TEXT,
				total_in_bed_time_milli       INTEGER,
				total_awake_time_milli        INTEGER,
				total_no_data_time_milli      INTEGER,
				total_light_sleep_time_milli  INTEGER,
				total_slow_wave_sleep_time_milli INTEGER,
				total_rem_sleep_time_milli    INTEGER,
				sleep_cycle_count             INTEGER,
				disturbance_count             INTEGER,
				baseline_sleep_needed_milli   INTEGER,
				need_from_sleep_debt_milli    INTEGER,
				need_from_recent_strain_milli INTEGER,
				need_from_recent_nap_milli    INTEGER,
				respiratory_rate              REAL,
				sleep_performance_pct         REAL,
				sleep_consistency_pct         REAL,
				sleep_efficiency_pct          REAL
			)`,
			`INSERT INTO sleep (
				id, user_id, created_at, updated_at, start, end, nap, score_state,
				total_in_bed_time_milli, total_awake_time_milli, total_no_data_time_milli,
				total_light_sleep_time_milli, total_slow_wave_sleep_time_milli, total_rem_sleep_time_milli,
				sleep_cycle_count, disturbance_count,
				baseline_sleep_needed_milli, need_from_sleep_debt_milli,
				need_from_recent_strain_milli, need_from_recent_nap_milli,
				respiratory_rate, sleep_performance_pct, sleep_consistency_pct, sleep_efficiency_pct
			)
			SELECT CAST(id AS TEXT), user_id, created_at, updated_at, start, end, nap, score_state,
				total_in_bed_time_milli, total_awake_time_milli, total_no_data_time_milli,
				total_light_sleep_time_milli, total_slow_wave_sleep_time_milli, total_rem_sleep_time_milli,
				sleep_cycle_count, disturbance_count,
				baseline_sleep_needed_milli, need_from_sleep_debt_milli,
				need_from_recent_strain_milli, need_from_recent_nap_milli,
				respiratory_rate, sleep_performance_pct, sleep_consistency_pct, sleep_efficiency_pct
			FROM sleep_old`,
			`DROP TABLE sleep_old`,

			`ALTER TABLE workout RENAME TO workout_old`,
			`CREATE TABLE workout (
				id                    TEXT PRIMARY KEY,
				user_id               INTEGER NOT NULL,
				created_at            TEXT NOT NULL,
				updated_at            TEXT NOT NULL,
				start                 TEXT NOT NULL,
				end                   TEXT,
				sport_id              INTEGER,
				score_state           TEXT,
				strain                REAL,
				average_heart_rate    INTEGER,
				max_heart_rate        INTEGER,
				kilojoule             REAL,
				percent_recorded      REAL,
				distance_meter        REAL,
				altitude_gain_meter   REAL,
				altitude_change_meter REAL,
				zone_zero_milli       INTEGER,
				zone_one_milli        INTEGER,
				zone_two_milli        INTEGER,
				zone_three_milli      INTEGER,
				zone_four_milli       INTEGER,
				zone_five_milli       INTEGER
			)`,
			`INSERT INTO workout (
				id, user_id, created_at, updated_at, start, end, sport_id, score_state,
				strain, average_heart_rate, max_heart_rate, kilojoule,
				percent_recorded, distance_meter, altitude_gain_meter, altitude_change_meter,
				zone_zero_milli, zone_one_milli, zone_two_milli,
				zone_three_milli, zone_four_milli, zone_five_milli
			)
			SELECT CAST(id AS TEXT), user_id, created_at, updated_at, start, end, sport_id, score_state,
				strain, average_heart_rate, max_heart_rate, kilojoule,
				percent_recorded, distance_meter, altitude_gain_meter, altitude_change_meter,
				zone_zero_milli, zone_one_milli, zone_two_milli,
				zone_three_milli, zone_four_milli, zone_five_milli
			FROM workout_old`,
			`DROP TABLE workout_old`,

			`CREATE INDEX IF NOT EXISTS idx_recovery_created ON recovery(created_at)`,
			`CREATE INDEX IF NOT EXISTS idx_sleep_start ON sleep(start)`,
			`CREATE INDEX IF NOT EXISTS idx_workout_start ON workout(start)`,
			`CREATE INDEX IF NOT EXISTS idx_recovery_scored_date ON recovery(score_state, created_at)`,
			`CREATE INDEX IF NOT EXISTS idx_cycle_scored_start ON cycle(score_state, start)`,
			`CREATE INDEX IF NOT EXISTS idx_sleep_scored ON sleep(nap, score_state, start)`,
			`CREATE INDEX IF NOT EXISTS idx_workout_scored_start ON workout(score_state, start)`,
		},
	},
	{
		version: 4,
		statements: []string{
			`CREATE VIEW IF NOT EXISTS daily_recovery AS
			SELECT
				date(created_at) AS day,
				recovery_score,
				hrv_rmssd,
				resting_heart_rate,
				spo2_percentage,
				skin_temp_celsius
			FROM recovery
			WHERE score_state = 'SCORED'`,

			`CREATE VIEW IF NOT EXISTS daily_sleep AS
			SELECT
				date(start) AS day,
				(total_in_bed_time_milli - total_awake_time_milli) / 3600000.0 AS actual_hours,
				total_in_bed_time_milli / 3600000.0 AS in_bed_hours,
				total_awake_time_milli / 3600000.0 AS awake_hours,
				(baseline_sleep_needed_milli + need_from_sleep_debt_milli + need_from_recent_strain_milli + need_from_recent_nap_milli) / 3600000.0 AS need_hours,
				((total_in_bed_time_milli - total_awake_time_milli) - (baseline_sleep_needed_milli + need_from_sleep_debt_milli + need_from_recent_strain_milli + need_from_recent_nap_milli)) / 3600000.0 AS need_gap_hours,
				sleep_efficiency_pct,
				sleep_performance_pct,
				sleep_consistency_pct,
				disturbance_count,
				sleep_cycle_count,
				respiratory_rate
			FROM sleep
			WHERE nap = 0 AND score_state = 'SCORED'`,

			`CREATE VIEW IF NOT EXISTS daily_strain AS
			SELECT
				date(start) AS day,
				strain,
				kilojoule,
				average_heart_rate,
				max_heart_rate
			FROM cycle
			WHERE score_state = 'SCORED'`,

			`CREATE VIEW IF NOT EXISTS workout_summary AS
			SELECT
				id,
				date(start) AS day,
				start,
				end,
				sport_id,
				(strftime('%s', end) - strftime('%s', start)) / 60.0 AS duration_minutes,
				strain,
				average_heart_rate,
				max_heart_rate,
				kilojoule,
				percent_recorded,
				distance_meter / 1000.0 AS distance_km,
				altitude_gain_meter,
				zone_zero_milli / 60000.0 AS zone_zero_minutes,
				zone_one_milli / 60000.0 AS zone_one_minutes,
				zone_two_milli / 60000.0 AS zone_two_minutes,
				zone_three_milli / 60000.0 AS zone_three_minutes,
				zone_four_milli / 60000.0 AS zone_four_minutes,
				zone_five_milli / 60000.0 AS zone_five_minutes
			FROM workout
			WHERE score_state = 'SCORED'`,
		},
	},
	{
		version: 5,
		statements: []string{
			`ALTER TABLE cycle ADD COLUMN timezone_offset TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE sleep ADD COLUMN timezone_offset TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE workout ADD COLUMN timezone_offset TEXT NOT NULL DEFAULT ''`,
			`DROP VIEW IF EXISTS daily_sleep`,
			`DROP VIEW IF EXISTS daily_strain`,
			`DROP VIEW IF EXISTS workout_summary`,
			`CREATE VIEW IF NOT EXISTS daily_sleep AS
			SELECT
				CASE WHEN IFNULL(timezone_offset,'') = '' THEN date(start) ELSE date(datetime(start, timezone_offset)) END AS day,
				(total_in_bed_time_milli - total_awake_time_milli) / 3600000.0 AS actual_hours,
				total_in_bed_time_milli / 3600000.0 AS in_bed_hours,
				total_awake_time_milli / 3600000.0 AS awake_hours,
				(baseline_sleep_needed_milli + need_from_sleep_debt_milli + need_from_recent_strain_milli + need_from_recent_nap_milli) / 3600000.0 AS need_hours,
				((total_in_bed_time_milli - total_awake_time_milli) - (baseline_sleep_needed_milli + need_from_sleep_debt_milli + need_from_recent_strain_milli + need_from_recent_nap_milli)) / 3600000.0 AS need_gap_hours,
				sleep_efficiency_pct,
				sleep_performance_pct,
				sleep_consistency_pct,
				disturbance_count,
				sleep_cycle_count,
				respiratory_rate
			FROM sleep
			WHERE nap = 0 AND score_state = 'SCORED'`,
			`CREATE VIEW IF NOT EXISTS daily_strain AS
			SELECT
				CASE WHEN IFNULL(timezone_offset,'') = '' THEN date(start) ELSE date(datetime(start, timezone_offset)) END AS day,
				strain,
				kilojoule,
				average_heart_rate,
				max_heart_rate
			FROM cycle
			WHERE score_state = 'SCORED'`,
			`CREATE VIEW IF NOT EXISTS workout_summary AS
			SELECT
				id,
				CASE WHEN IFNULL(timezone_offset,'') = '' THEN date(start) ELSE date(datetime(start, timezone_offset)) END AS day,
				start,
				end,
				sport_id,
				(strftime('%s', end) - strftime('%s', start)) / 60.0 AS duration_minutes,
				strain,
				average_heart_rate,
				max_heart_rate,
				kilojoule,
				percent_recorded,
				distance_meter / 1000.0 AS distance_km,
				altitude_gain_meter,
				zone_zero_milli / 60000.0 AS zone_zero_minutes,
				zone_one_milli / 60000.0 AS zone_one_minutes,
				zone_two_milli / 60000.0 AS zone_two_minutes,
				zone_three_milli / 60000.0 AS zone_three_minutes,
				zone_four_milli / 60000.0 AS zone_four_minutes,
				zone_five_milli / 60000.0 AS zone_five_minutes
			FROM workout
			WHERE score_state = 'SCORED'`,
		},
	},
	{
		// Sleep duration should exclude no-data gaps (WHOOP stage_summary).
		// Recreate daily_sleep while preserving timezone-aware day expression from v5.
		version: 6,
		statements: []string{
			`DROP VIEW IF EXISTS daily_sleep`,
			`CREATE VIEW IF NOT EXISTS daily_sleep AS
			SELECT
				CASE WHEN IFNULL(timezone_offset,'') = '' THEN date(start) ELSE date(datetime(start, timezone_offset)) END AS day,
				(total_in_bed_time_milli - total_awake_time_milli - COALESCE(total_no_data_time_milli, 0)) / 3600000.0 AS actual_hours,
				total_in_bed_time_milli / 3600000.0 AS in_bed_hours,
				total_awake_time_milli / 3600000.0 AS awake_hours,
				(baseline_sleep_needed_milli + need_from_sleep_debt_milli + need_from_recent_strain_milli + need_from_recent_nap_milli) / 3600000.0 AS need_hours,
				((total_in_bed_time_milli - total_awake_time_milli - COALESCE(total_no_data_time_milli, 0)) - (baseline_sleep_needed_milli + need_from_sleep_debt_milli + need_from_recent_strain_milli + need_from_recent_nap_milli)) / 3600000.0 AS need_gap_hours,
				sleep_efficiency_pct,
				sleep_performance_pct,
				sleep_consistency_pct,
				disturbance_count,
				sleep_cycle_count,
				respiratory_rate
			FROM sleep
			WHERE nap = 0 AND score_state = 'SCORED'`,
		},
	},
}

func (db *DB) migrate() error {
	// Create schema_version table to track applied migrations
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY
	)`); err != nil {
		return fmt.Errorf("create schema_version table: %w", err)
	}

	currentVersion := 0
	err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&currentVersion)
	if err == sql.ErrNoRows {
		currentVersion = 0
	} else if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	for _, m := range migrations {
		if m.version <= currentVersion {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration v%d: %w", m.version, err)
		}

		for _, stmt := range m.statements {
			if _, err := tx.Exec(stmt); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("migration v%d: %w", m.version, err)
			}
		}

		if _, err := tx.Exec(`INSERT INTO schema_version (version) VALUES (?)`, m.version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration v%d: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration v%d: %w", m.version, err)
		}
	}
	return nil
}
