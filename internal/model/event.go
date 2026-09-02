package model

import (
	"context"
	"database/sql"
	"fmt"
)

type Event struct {
	ID            int64
	Kind          string
	Status        string
	Title         string
	StartDate     sql.NullString
	EndDate       sql.NullString
	TimeframeNote string
	Notes         string
	CreatedAt     string
	People        []Person // populated by ListEventsNeedingAttention
}

// CreateEvent inserts a new event and returns its id.
func (s *Store) CreateEvent(ctx context.Context, e Event) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO events (kind, status, title, start_date, end_date, timeframe_note, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, e.Kind, e.Status, e.Title, e.StartDate, e.EndDate, e.TimeframeNote, e.Notes)
	if err != nil {
		return 0, fmt.Errorf("create event: %w", err)
	}
	return res.LastInsertId()
}

// UpdateEvent updates an event's core fields (not its participants).
func (s *Store) UpdateEvent(ctx context.Context, e Event) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE events SET kind = ?, status = ?, title = ?, start_date = ?, end_date = ?,
		       timeframe_note = ?, notes = ?
		WHERE id = ?
	`, e.Kind, e.Status, e.Title, e.StartDate, e.EndDate, e.TimeframeNote, e.Notes, e.ID)
	if err != nil {
		return fmt.Errorf("update event: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update event rows affected: %w", err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteEvent removes an event and (via ON DELETE CASCADE) its participant
// rows and any reminders linked to it.
func (s *Store) DeleteEvent(ctx context.Context, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM events WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete event: %w", err)
	}
	return nil
}

// GetEvent loads an event's core fields only (no participants). Returns
// (nil, nil) if no event with that id exists.
func (s *Store) GetEvent(ctx context.Context, id int64) (*Event, error) {
	var e Event
	err := s.db.QueryRowContext(ctx, `
		SELECT id, kind, status, title, start_date, end_date, timeframe_note, notes, created_at
		FROM events WHERE id = ?
	`, id).Scan(&e.ID, &e.Kind, &e.Status, &e.Title, &e.StartDate, &e.EndDate, &e.TimeframeNote, &e.Notes, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}
	return &e, nil
}

// GetEventDetail loads an event's core fields plus its participants. Returns
// (nil, nil) if no event with that id exists.
func (s *Store) GetEventDetail(ctx context.Context, id int64) (*Event, error) {
	e, err := s.GetEvent(ctx, id)
	if err != nil || e == nil {
		return e, err
	}
	people, err := s.ListEventPeople(ctx, id)
	if err != nil {
		return nil, err
	}
	e.People = people
	return e, nil
}

// ListEvents returns every event with its participants populated, ordered by
// status priority (idea, tentative, confirmed, then done/cancelled/anything
// else) and soonest-dated first within each group. The Events page groups
// these into sections by iterating the already-sorted result.
func (s *Store) ListEvents(ctx context.Context) ([]Event, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, status, title, start_date, end_date, timeframe_note, notes, created_at
		FROM events
		ORDER BY CASE LOWER(status)
		             WHEN 'idea' THEN 0
		             WHEN 'tentative' THEN 1
		             WHEN 'confirmed' THEN 2
		             WHEN 'done' THEN 3
		             WHEN 'cancelled' THEN 4
		             ELSE 5
		         END, start_date IS NULL, start_date
	`)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Kind, &e.Status, &e.Title, &e.StartDate, &e.EndDate, &e.TimeframeNote, &e.Notes, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range events {
		people, err := s.ListEventPeople(ctx, events[i].ID)
		if err != nil {
			return nil, err
		}
		events[i].People = people
	}
	return events, nil
}

// AddPersonToEvent adds a person as a participant in an event.
func (s *Store) AddPersonToEvent(ctx context.Context, eventID, personID int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO event_people (event_id, person_id) VALUES (?, ?)
	`, eventID, personID)
	if err != nil {
		return fmt.Errorf("add person to event: %w", err)
	}
	return nil
}

// RemovePersonFromEvent removes a person as a participant in an event.
func (s *Store) RemovePersonFromEvent(ctx context.Context, eventID, personID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM event_people WHERE event_id = ? AND person_id = ?
	`, eventID, personID)
	if err != nil {
		return fmt.Errorf("remove person from event: %w", err)
	}
	return nil
}

// ListEventsNeedingAttention returns events that need a look: any event
// still in "tentative" status (it needs confirming, regardless of date) plus
// any "confirmed" event starting within the next withinDays days. Ordered
// soonest-dated first, with undated (tentative, no date yet) events last.
func (s *Store) ListEventsNeedingAttention(ctx context.Context, withinDays int) ([]Event, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, status, title, start_date, end_date, timeframe_note, notes, created_at
		FROM events
		WHERE LOWER(status) = 'tentative'
		   OR (LOWER(status) = 'confirmed' AND start_date IS NOT NULL
		       AND date(start_date) BETWEEN date('now') AND date('now', '+' || ? || ' days'))
		ORDER BY start_date IS NULL, start_date
	`, withinDays)
	if err != nil {
		return nil, fmt.Errorf("list events needing attention: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Kind, &e.Status, &e.Title, &e.StartDate, &e.EndDate, &e.TimeframeNote, &e.Notes, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range events {
		people, err := s.ListEventPeople(ctx, events[i].ID)
		if err != nil {
			return nil, err
		}
		events[i].People = people
	}
	return events, nil
}

// ListEventPeople returns the participants of an event.
func (s *Store) ListEventPeople(ctx context.Context, eventID int64) ([]Person, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.first_name, p.last_name, p.nickname
		FROM event_people ep
		JOIN people p ON p.id = ep.person_id
		WHERE ep.event_id = ?
		ORDER BY p.last_name, p.first_name
	`, eventID)
	if err != nil {
		return nil, fmt.Errorf("list event people: %w", err)
	}
	defer rows.Close()

	var people []Person
	for rows.Next() {
		var p Person
		if err := rows.Scan(&p.ID, &p.FirstName, &p.LastName, &p.Nickname); err != nil {
			return nil, fmt.Errorf("scan event person: %w", err)
		}
		people = append(people, p)
	}
	return people, rows.Err()
}
