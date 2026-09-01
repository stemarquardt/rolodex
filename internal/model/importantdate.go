package model

import (
	"context"
	"database/sql"
	"fmt"
)

type ImportantDate struct {
	ID    int64
	Type  string
	Label string
	Month int
	Day   int
	Year  sql.NullInt64 // annually-recurring dates (birthdays) leave this null
}

func (s *Store) ListImportantDates(ctx context.Context, personID int64) ([]ImportantDate, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, label, month, day, year
		FROM important_dates WHERE person_id = ? ORDER BY month, day
	`, personID)
	if err != nil {
		return nil, fmt.Errorf("list important dates: %w", err)
	}
	defer rows.Close()

	var items []ImportantDate
	for rows.Next() {
		var d ImportantDate
		if err := rows.Scan(&d.ID, &d.Type, &d.Label, &d.Month, &d.Day, &d.Year); err != nil {
			return nil, fmt.Errorf("scan important date: %w", err)
		}
		items = append(items, d)
	}
	return items, rows.Err()
}
