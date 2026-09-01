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
