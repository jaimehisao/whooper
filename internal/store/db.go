package store

import (
	"database/sql"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(wal)&_pragma=synchronous(normal)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	// SQLite allows only one writer. Cap the pool so parallel sync goroutines
	// serialize on the same connection instead of racing to lock timeouts.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	store := &DB{db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func OpenReadOnly(path string) (*DB, error) {
	dsn := url.URL{Scheme: "file", Path: path}
	query := dsn.Query()
	query.Set("mode", "ro")
	query.Set("_pragma", "busy_timeout(5000)")
	dsn.RawQuery = query.Encode()

	db, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &DB{db}, nil
}
