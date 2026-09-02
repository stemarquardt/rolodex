package model

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Reminder struct {
	ID                 int64
	PersonID           sql.NullInt64
	PersonName         string // populated by joins below; empty if not tied to a person
	EventID            sql.NullInt64
	EventTitle         string // populated by ListReminders via join; empty if not tied to an event
	DueDate            string
	RecurrenceInterval sql.NullInt64
	RecurrenceUnit     sql.NullString
	Note               string
	Status             string
}

func (r Reminder) Recurring() bool {
	return r.RecurrenceInterval.Valid && r.RecurrenceUnit.Valid
}

const reminderSelectColumns = `
	r.id, r.person_id, p.first_name, p.last_name, r.event_id, e.title,
	r.due_date, r.recurrence_interval, r.recurrence_unit, r.note, r.status
`

func scanReminder(row interface{ Scan(...any) error }) (Reminder, error) {
	var r Reminder
	var firstName, lastName, eventTitle sql.NullString
	if err := row.Scan(&r.ID, &r.PersonID, &firstName, &lastName, &r.EventID, &eventTitle,
		&r.DueDate, &r.RecurrenceInterval, &r.RecurrenceUnit, &r.Note, &r.Status); err != nil {
		return Reminder{}, fmt.Errorf("scan reminder: %w", err)
	}
	r.PersonName = strings.TrimSpace(firstName.String + " " + lastName.String)
	r.EventTitle = eventTitle.String
	return r, nil
}

// ListDueReminders returns pending reminders due today or earlier (overdue),
// soonest first, each annotated with the linked person's name (if any).
func (s *Store) ListDueReminders(ctx context.Context) ([]Reminder, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+reminderSelectColumns+`
		FROM reminders r
		LEFT JOIN people p ON p.id = r.person_id
		LEFT JOIN events e ON e.id = r.event_id
		WHERE r.status = 'pending' AND date(r.due_date) <= date('now')
		ORDER BY r.due_date
	`)
	if err != nil {
		return nil, fmt.Errorf("list due reminders: %w", err)
	}
	defer rows.Close()

	var reminders []Reminder
	for rows.Next() {
		r, err := scanReminder(rows)
		if err != nil {
			return nil, err
		}
		reminders = append(reminders, r)
	}
	return reminders, rows.Err()
}

// ListReminders returns every reminder (pending and done), soonest-due
// first, each annotated with linked person/event names. Powers the
// standalone Reminders page.
func (s *Store) ListReminders(ctx context.Context) ([]Reminder, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+reminderSelectColumns+`
		FROM reminders r
		LEFT JOIN people p ON p.id = r.person_id
		LEFT JOIN events e ON e.id = r.event_id
		ORDER BY r.status = 'done', r.due_date
	`)
	if err != nil {
		return nil, fmt.Errorf("list reminders: %w", err)
	}
	defer rows.Close()

	var reminders []Reminder
	for rows.Next() {
		r, err := scanReminder(rows)
		if err != nil {
			return nil, err
		}
		reminders = append(reminders, r)
	}
	return reminders, rows.Err()
}

// ListRemindersForPerson returns every reminder tied to a specific person
// (pending and done), for that person's profile page.
func (s *Store) ListRemindersForPerson(ctx context.Context, personID int64) ([]Reminder, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+reminderSelectColumns+`
		FROM reminders r
		LEFT JOIN people p ON p.id = r.person_id
		LEFT JOIN events e ON e.id = r.event_id
		WHERE r.person_id = ?
		ORDER BY r.status = 'done', r.due_date
	`, personID)
	if err != nil {
		return nil, fmt.Errorf("list reminders for person: %w", err)
	}
	defer rows.Close()

	var reminders []Reminder
	for rows.Next() {
		r, err := scanReminder(rows)
		if err != nil {
			return nil, err
		}
		reminders = append(reminders, r)
	}
	return reminders, rows.Err()
}

// GetReminder loads a single reminder's core fields (no name joins).
// Returns (nil, nil) if no reminder with that id exists.
func (s *Store) GetReminder(ctx context.Context, id int64) (*Reminder, error) {
	var r Reminder
	err := s.db.QueryRowContext(ctx, `
		SELECT id, person_id, event_id, due_date, recurrence_interval, recurrence_unit, note, status
		FROM reminders WHERE id = ?
	`, id).Scan(&r.ID, &r.PersonID, &r.EventID, &r.DueDate, &r.RecurrenceInterval, &r.RecurrenceUnit, &r.Note, &r.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get reminder: %w", err)
	}
	return &r, nil
}

// CreateReminder inserts a new reminder and returns its id. PersonID/EventID
// may be left invalid (NULL) for a standalone reminder not tied to either.
func (s *Store) CreateReminder(ctx context.Context, r Reminder) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO reminders (person_id, event_id, due_date, recurrence_interval, recurrence_unit, note, status)
		VALUES (?, ?, ?, ?, ?, ?, 'pending')
	`, r.PersonID, r.EventID, r.DueDate, r.RecurrenceInterval, r.RecurrenceUnit, r.Note)
	if err != nil {
		return 0, fmt.Errorf("create reminder: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) DeleteReminder(ctx context.Context, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM reminders WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete reminder: %w", err)
	}
	return nil
}

// CompleteReminder marks a reminder done. If it's recurring, instead of
// disappearing it advances due_date by one interval and stays pending — see
// PLANNING.md: "When a recurring reminder is completed, compute the next
// due_date rather than building a full RRULE engine."
func (s *Store) CompleteReminder(ctx context.Context, id int64) error {
	r, err := s.GetReminder(ctx, id)
	if err != nil {
		return err
	}
	if r == nil {
		return sql.ErrNoRows
	}

	if !r.Recurring() {
		if _, err := s.db.ExecContext(ctx, `UPDATE reminders SET status = 'done' WHERE id = ?`, id); err != nil {
			return fmt.Errorf("complete reminder: %w", err)
		}
		return nil
	}

	due, err := time.Parse("2006-01-02", r.DueDate)
	if err != nil {
		return fmt.Errorf("parse due date %q: %w", r.DueDate, err)
	}
	next := nextDueDate(due, int(r.RecurrenceInterval.Int64), r.RecurrenceUnit.String)
	if _, err := s.db.ExecContext(ctx, `UPDATE reminders SET due_date = ? WHERE id = ?`, next.Format("2006-01-02"), id); err != nil {
		return fmt.Errorf("advance recurring reminder: %w", err)
	}
	return nil
}

// nextDueDate advances due by interval units of unit ("days", "weeks", or
// "months").
func nextDueDate(due time.Time, interval int, unit string) time.Time {
	switch unit {
	case "weeks":
		return due.AddDate(0, 0, interval*7)
	case "months":
		return due.AddDate(0, interval, 0)
	default: // "days"
		return due.AddDate(0, 0, interval)
	}
}
