package model

import (
	"context"
	"database/sql"
	"fmt"
)

type Circle struct {
	ID          int64
	Name        string
	Description string
}

// CircleMembership is a person's membership in one circle, including the
// optional per-membership context note (e.g. "Met at the bouldering wall").
type CircleMembership struct {
	CircleID   int64
	CircleName string
	Note       sql.NullString
}

func (s *Store) ListCircleMemberships(ctx context.Context, personID int64) ([]CircleMembership, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.name, pc.note
		FROM person_circles pc
		JOIN circles c ON c.id = pc.circle_id
		WHERE pc.person_id = ?
		ORDER BY c.name
	`, personID)
	if err != nil {
		return nil, fmt.Errorf("list circle memberships: %w", err)
	}
	defer rows.Close()

	var items []CircleMembership
	for rows.Next() {
		var m CircleMembership
		if err := rows.Scan(&m.CircleID, &m.CircleName, &m.Note); err != nil {
			return nil, fmt.Errorf("scan circle membership: %w", err)
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

// ListCircles returns every circle, for populating an autocomplete/datalist
// when adding a person to a circle. Full Circle list/detail pages are out of
// scope for now — this is just enough to support that one workflow.
func (s *Store) ListCircles(ctx context.Context) ([]Circle, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, description FROM circles ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list circles: %w", err)
	}
	defer rows.Close()

	var items []Circle
	for rows.Next() {
		var c Circle
		if err := rows.Scan(&c.ID, &c.Name, &c.Description); err != nil {
			return nil, fmt.Errorf("scan circle: %w", err)
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

// GetOrCreateCircleByName finds a circle by name (case-insensitive) or
// creates it if it doesn't exist yet, so typing a new circle name into the
// "add to circle" form just works without a separate create-circle step.
func (s *Store) GetOrCreateCircleByName(ctx context.Context, name string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM circles WHERE name = ? COLLATE NOCASE
	`, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("lookup circle: %w", err)
	}

	res, err := s.db.ExecContext(ctx, `INSERT INTO circles (name) VALUES (?)`, name)
	if err != nil {
		return 0, fmt.Errorf("create circle: %w", err)
	}
	return res.LastInsertId()
}

// AddPersonToCircle adds (or updates the note on) a person's membership in a
// circle.
func (s *Store) AddPersonToCircle(ctx context.Context, personID, circleID int64, note string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO person_circles (person_id, circle_id, note) VALUES (?, ?, ?)
		ON CONFLICT (person_id, circle_id) DO UPDATE SET note = excluded.note
	`, personID, circleID, note)
	if err != nil {
		return fmt.Errorf("add person to circle: %w", err)
	}
	return nil
}

func (s *Store) RemovePersonFromCircle(ctx context.Context, personID, circleID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM person_circles WHERE person_id = ? AND circle_id = ?
	`, personID, circleID)
	if err != nil {
		return fmt.Errorf("remove person from circle: %w", err)
	}
	return nil
}
