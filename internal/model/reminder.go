package model

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Reminder struct {
	ID                 int64
	PersonID           sql.NullInt64
	PersonName         string // populated by ListDueReminders via join; empty if not tied to a person
	EventID            sql.NullInt64
	DueDate            string
	RecurrenceInterval sql.NullInt64
	RecurrenceUnit     sql.NullString
	Note               string
	Status             string
}

func (r Reminder) Recurring() bool {
	return r.RecurrenceInterval.Valid && r.RecurrenceUnit.Valid
}

// ListDueReminders returns pending reminders due today or earlier (overdue),
// soonest first, each annotated with the linked person's name (if any).
func (s *Store) ListDueReminders(ctx context.Context) ([]Reminder, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.person_id, p.first_name, p.last_name, r.event_id,
		       r.due_date, r.recurrence_interval, r.recurrence_unit, r.note, r.status
		FROM reminders r
		LEFT JOIN people p ON p.id = r.person_id
		WHERE r.status = 'pending' AND date(r.due_date) <= date('now')
		ORDER BY r.due_date
	`)
	if err != nil {
		return nil, fmt.Errorf("list due reminders: %w", err)
	}
	defer rows.Close()

	var reminders []Reminder
	for rows.Next() {
		var r Reminder
		var firstName, lastName sql.NullString
		if err := rows.Scan(&r.ID, &r.PersonID, &firstName, &lastName, &r.EventID,
			&r.DueDate, &r.RecurrenceInterval, &r.RecurrenceUnit, &r.Note, &r.Status); err != nil {
			return nil, fmt.Errorf("scan reminder: %w", err)
		}
		r.PersonName = strings.TrimSpace(firstName.String + " " + lastName.String)
		reminders = append(reminders, r)
	}
	return reminders, rows.Err()
}
