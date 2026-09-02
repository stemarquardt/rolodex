package model

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
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

// CreateImportantDate inserts a new important date. Pass year=nil for an
// annually-recurring date (birthday/anniversary with no known year).
func (s *Store) CreateImportantDate(ctx context.Context, personID int64, typ, label string, month, day int, year *int) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO important_dates (person_id, type, label, month, day, year) VALUES (?, ?, ?, ?, ?, ?)
	`, personID, typ, label, month, day, year)
	if err != nil {
		return 0, fmt.Errorf("create important date: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) DeleteImportantDate(ctx context.Context, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM important_dates WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete important date: %w", err)
	}
	return nil
}

// UpcomingImportantDate is an ImportantDate whose next annual occurrence has
// been resolved against today's date, for cross-person display on the Today
// page (ListImportantDates only ever queries one person at a time).
type UpcomingImportantDate struct {
	ImportantDate
	PersonID   int64
	PersonName string
	DaysUntil  int
}

// ListUpcomingImportantDates returns important dates across all people whose
// next annual occurrence falls within the next withinDays days (0 = today),
// soonest first. The month/day recur every year regardless of the stored
// `year` value, so the "next occurrence" is resolved in Go rather than SQL.
func (s *Store) ListUpcomingImportantDates(ctx context.Context, withinDays int) ([]UpcomingImportantDate, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT d.id, d.type, d.label, d.month, d.day, d.year,
		       p.id, p.first_name, p.last_name, p.nickname
		FROM important_dates d
		JOIN people p ON p.id = d.person_id
	`)
	if err != nil {
		return nil, fmt.Errorf("list upcoming important dates: %w", err)
	}
	defer rows.Close()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var upcoming []UpcomingImportantDate
	for rows.Next() {
		var d ImportantDate
		var p Person
		if err := rows.Scan(&d.ID, &d.Type, &d.Label, &d.Month, &d.Day, &d.Year,
			&p.ID, &p.FirstName, &p.LastName, &p.Nickname); err != nil {
			return nil, fmt.Errorf("scan upcoming important date: %w", err)
		}

		next := nextOccurrence(today, d.Month, d.Day)
		daysUntil := int(next.Sub(today).Hours() / 24)
		if daysUntil > withinDays {
			continue
		}
		upcoming = append(upcoming, UpcomingImportantDate{
			ImportantDate: d,
			PersonID:      p.ID,
			PersonName:    p.FullName(),
			DaysUntil:     daysUntil,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(upcoming, func(i, j int) bool { return upcoming[i].DaysUntil < upcoming[j].DaysUntil })
	return upcoming, nil
}

// nextOccurrence returns the next date on/after today with the given
// month/day, rolling to next year if this year's occurrence already passed.
func nextOccurrence(today time.Time, month, day int) time.Time {
	candidate := time.Date(today.Year(), time.Month(month), day, 0, 0, 0, 0, today.Location())
	if candidate.Before(today) {
		candidate = time.Date(today.Year()+1, time.Month(month), day, 0, 0, 0, 0, today.Location())
	}
	return candidate
}
