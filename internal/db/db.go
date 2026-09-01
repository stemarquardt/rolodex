package db

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// Open opens the SQLite database at path, enables foreign keys, and applies
// the schema (idempotent — safe to run on every startup).
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)", path)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite only supports one writer at a time; a single connection avoids
	// SQLITE_BUSY errors under concurrent requests without needing a pool.
	sqlDB.SetMaxOpenConns(1)

	if _, err := sqlDB.Exec(schemaSQL); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	if err := seed(sqlDB); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("seed: %w", err)
	}

	return sqlDB, nil
}
