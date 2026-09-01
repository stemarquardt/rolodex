// Package model provides the data-access layer: plain SQL queries against
// the SQLite schema in internal/db/schema.sql, no ORM. The dataset is
// personal-scale, so straightforward per-entity queries are preferred over
// complex joins or batching machinery.
package model

import "database/sql"

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}
