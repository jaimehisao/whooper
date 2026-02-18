package store

func (db *DB) migrate() error {
	statements := []string{
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
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
