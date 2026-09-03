package model

import (
	"context"
	"fmt"
)

// EventNote is a timestamped freeform log entry on an Event, mirroring Note
// (person profiles) — see internal/db/schema.sql's event_notes table
// comment for why this is a separate table rather than a generalized one.
type EventNote struct {
	ID        int64
	Body      string
	CreatedAt string
}

func (s *Store) ListEventNotes(ctx context.Context, eventID int64) ([]EventNote, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, body, created_at FROM event_notes WHERE event_id = ? ORDER BY created_at DESC
	`, eventID)
	if err != nil {
		return nil, fmt.Errorf("list event notes: %w", err)
	}
	defer rows.Close()

	var items []EventNote
	for rows.Next() {
		var n EventNote
		if err := rows.Scan(&n.ID, &n.Body, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan event note: %w", err)
		}
		items = append(items, n)
	}
	return items, rows.Err()
}

func (s *Store) CreateEventNote(ctx context.Context, eventID int64, body string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO event_notes (event_id, body) VALUES (?, ?)
	`, eventID, body)
	if err != nil {
		return 0, fmt.Errorf("create event note: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) DeleteEventNote(ctx context.Context, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM event_notes WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete event note: %w", err)
	}
	return nil
}
