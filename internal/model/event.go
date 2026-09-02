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

// ListEventsNeedingAttention returns events that need a look: any event
// still in "tentative" status (it needs confirming, regardless of date) plus
// any "confirmed" event starting within the next withinDays days. Ordered
// soonest-dated first, with undated (tentative, no date yet) events last.
func (s *Store) ListEventsNeedingAttention(ctx context.Context, withinDays int) ([]Event, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, status, title, start_date, end_date, timeframe_note, notes, created_at
		FROM events
		WHERE status = 'tentative'
		   OR (status = 'confirmed' AND start_date IS NOT NULL
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
		people, err := s.listEventPeople(ctx, events[i].ID)
		if err != nil {
			return nil, err
		}
		events[i].People = people
	}
	return events, nil
}

func (s *Store) listEventPeople(ctx context.Context, eventID int64) ([]Person, error) {
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
