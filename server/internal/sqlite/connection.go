// Package sqlite is the analytical read layer: CRM/BI queries run here,
// populated from Badger by the sync pipeline. Pure-Go driver (no Cgo).
package sqlite

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

type DB struct {
	sql *sql.DB
}

func Open(path string) (*DB, error) {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// modernc sqlite: single writer is fine for our volume
	db.SetMaxOpenConns(4)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating sqlite: %w", err)
	}
	return &DB{sql: db}, nil
}

func (d *DB) Close() error { return d.sql.Close() }

func (d *DB) Exec(query string, args ...any) (sql.Result, error) {
	return d.sql.Exec(query, args...)
}

func (d *DB) Query(query string, args ...any) (*sql.Rows, error) {
	return d.sql.Query(query, args...)
}

func (d *DB) QueryRow(query string, args ...any) *sql.Row {
	return d.sql.QueryRow(query, args...)
}
