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
	MemberCount int // populated by ListCircles only
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

// ListCircles returns every circle with its member count, for both the
// Circles list page and the add-to-circle datalist on a person's profile
// (which only reads .Name).
func (s *Store) ListCircles(ctx context.Context) ([]Circle, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.name, c.description, COUNT(pc.person_id)
		FROM circles c
		LEFT JOIN person_circles pc ON pc.circle_id = c.id
		GROUP BY c.id
		ORDER BY c.name
	`)
	if err != nil {
		return nil, fmt.Errorf("list circles: %w", err)
	}
	defer rows.Close()

	var items []Circle
	for rows.Next() {
		var c Circle
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.MemberCount); err != nil {
			return nil, fmt.Errorf("scan circle: %w", err)
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

// CreateCircle inserts a new circle with an explicit name and description,
// for the Circles page's "+ New circle" form. (GetOrCreateCircleByName below
// is the separate, name-only path used by the quick "add this person to a
// circle" form on a person's profile.)
func (s *Store) CreateCircle(ctx context.Context, name, description string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO circles (name, description) VALUES (?, ?)`, name, description)
	if err != nil {
		return 0, fmt.Errorf("create circle: %w", err)
	}
	return res.LastInsertId()
}

// GetCircle loads a circle's core fields only (no members). Returns (nil,
// nil) if no circle with that id exists.
func (s *Store) GetCircle(ctx context.Context, id int64) (*Circle, error) {
	var c Circle
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description FROM circles WHERE id = ?
	`, id).Scan(&c.ID, &c.Name, &c.Description)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get circle: %w", err)
	}
	return &c, nil
}

// UpdateCircle updates a circle's name and description.
func (s *Store) UpdateCircle(ctx context.Context, c Circle) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE circles SET name = ?, description = ? WHERE id = ?
	`, c.Name, c.Description, c.ID)
	if err != nil {
		return fmt.Errorf("update circle: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update circle rows affected: %w", err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteCircle removes a circle and (via ON DELETE CASCADE) its memberships.
func (s *Store) DeleteCircle(ctx context.Context, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM circles WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete circle: %w", err)
	}
	return nil
}

// CircleMember is a person belonging to a circle, from the circle's point of
// view — the reverse direction of CircleMembership (person -> circles).
type CircleMember struct {
	PersonID   int64
	PersonName string
	Note       sql.NullString
}

func (s *Store) ListCircleMembers(ctx context.Context, circleID int64) ([]CircleMember, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.first_name, p.last_name, p.nickname, pc.note
		FROM person_circles pc
		JOIN people p ON p.id = pc.person_id
		WHERE pc.circle_id = ?
		ORDER BY p.last_name, p.first_name
	`, circleID)
	if err != nil {
		return nil, fmt.Errorf("list circle members: %w", err)
	}
	defer rows.Close()

	var members []CircleMember
	for rows.Next() {
		var p Person
		var m CircleMember
		if err := rows.Scan(&p.ID, &p.FirstName, &p.LastName, &p.Nickname, &m.Note); err != nil {
			return nil, fmt.Errorf("scan circle member: %w", err)
		}
		m.PersonID = p.ID
		m.PersonName = p.FullName()
		members = append(members, m)
	}
	return members, rows.Err()
}

// CircleDetail is the full circle view: a Circle plus its members.
type CircleDetail struct {
	Circle
	Members []CircleMember
}

// GetCircleDetail loads a circle's core fields plus its members. Returns
// (nil, nil) if no circle with that id exists.
func (s *Store) GetCircleDetail(ctx context.Context, id int64) (*CircleDetail, error) {
	c, err := s.GetCircle(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}

	members, err := s.ListCircleMembers(ctx, id)
	if err != nil {
		return nil, err
	}

	return &CircleDetail{Circle: *c, Members: members}, nil
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
