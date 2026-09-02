package model

import (
	"context"
	"fmt"
)

// OptionValue is one entry in a managed suggestion list backing a <select>
// field elsewhere in the app (Event kind/status, ContactInfo/ImportantDate
// type). Deliberately not referenced by a foreign key from those tables —
// see internal/db/schema.sql's option_values table comment.
type OptionValue struct {
	ID       int64
	Category string
	Value    string
}

// Category values — must match the keys in internal/db/seed.go's
// defaultOptionValues.
const (
	CategoryEventKind         = "event_kind"
	CategoryEventStatus       = "event_status"
	CategoryContactInfoType   = "contact_info_type"
	CategoryImportantDateType = "important_date_type"
)

// ListOptionValues returns a category's values in insertion order, so a
// deliberately-ordered seed list (e.g. event_status's idea -> ... ->
// cancelled progression) stays in that order as admin-added values append
// after it.
func (s *Store) ListOptionValues(ctx context.Context, category string) ([]OptionValue, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, category, value FROM option_values WHERE category = ? ORDER BY id
	`, category)
	if err != nil {
		return nil, fmt.Errorf("list option values: %w", err)
	}
	defer rows.Close()

	var items []OptionValue
	for rows.Next() {
		var v OptionValue
		if err := rows.Scan(&v.ID, &v.Category, &v.Value); err != nil {
			return nil, fmt.Errorf("scan option value: %w", err)
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

// ListAllOptionValues returns every category's values in one query, keyed
// by category — used by the Settings page, which manages all of them at
// once.
func (s *Store) ListAllOptionValues(ctx context.Context) (map[string][]OptionValue, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, category, value FROM option_values ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("list all option values: %w", err)
	}
	defer rows.Close()

	byCategory := make(map[string][]OptionValue)
	for rows.Next() {
		var v OptionValue
		if err := rows.Scan(&v.ID, &v.Category, &v.Value); err != nil {
			return nil, fmt.Errorf("scan option value: %w", err)
		}
		byCategory[v.Category] = append(byCategory[v.Category], v)
	}
	return byCategory, rows.Err()
}

// CreateOptionValue adds a new value to a category's suggestion list. Fails
// on a case-insensitive duplicate within the same category (the schema's
// UNIQUE(category, value COLLATE NOCASE) constraint).
func (s *Store) CreateOptionValue(ctx context.Context, category, value string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO option_values (category, value) VALUES (?, ?)
	`, category, value)
	if err != nil {
		return 0, fmt.Errorf("create option value: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) DeleteOptionValue(ctx context.Context, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM option_values WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete option value: %w", err)
	}
	return nil
}
