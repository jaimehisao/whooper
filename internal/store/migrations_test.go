package store

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestMigrate_UpgradesFromVersionOne(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "upgrade.db")
	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	if _, err := raw.Exec(`CREATE TABLE schema_version (version INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create schema_version: %v", err)
	}
	for _, stmt := range migrations[0].statements {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatalf("seed v1 migration statement: %v", err)
		}
	}
	if _, err := raw.Exec(`INSERT INTO schema_version (version) VALUES (1)`); err != nil {
		t.Fatalf("record v1 migration: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("close seed DB: %v", err)
	}

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open upgraded DB: %v", err)
	}
	defer db.Close()

	assertSchemaVersion(t, db, 2)
	for _, name := range []string{
		"idx_recovery_scored_date",
		"idx_cycle_scored_start",
		"idx_sleep_scored",
		"idx_workout_scored_start",
	} {
		assertIndexExists(t, db, name)
	}
}

func TestMigrate_IdempotentOnReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "idempotent.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	assertSchemaVersion(t, db, 2)
	if err := db.Close(); err != nil {
		t.Fatalf("close first DB: %v", err)
	}

	db, err = Open(dbPath)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer db.Close()

	assertSchemaVersion(t, db, 2)
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_version`).Scan(&count); err != nil {
		t.Fatalf("count schema_version rows: %v", err)
	}
	if count != 2 {
		t.Fatalf("schema_version rows = %d, want 2", count)
	}
}

func TestMigrate_CreatesExpectedIndexes(t *testing.T) {
	db := openTestDB(t)

	for _, name := range []string{
		"idx_cycle_start",
		"idx_recovery_created",
		"idx_sleep_start",
		"idx_workout_start",
		"idx_recovery_scored_date",
		"idx_cycle_scored_start",
		"idx_sleep_scored",
		"idx_workout_scored_start",
	} {
		assertIndexExists(t, db, name)
	}
}

func assertSchemaVersion(t *testing.T, db *DB, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&got); err != nil {
		t.Fatalf("read schema version: %v", err)
	}
	if got != want {
		t.Fatalf("schema version = %d, want %d", got, want)
	}
}

func assertIndexExists(t *testing.T, db *DB, name string) {
	t.Helper()
	var got string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='index' AND name=?`, name).Scan(&got)
	if err != nil {
		t.Fatalf("index %q not found: %v", name, err)
	}
}
