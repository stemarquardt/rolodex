package model

import (
	"context"
	"fmt"
)

type Note struct {
	ID        int64
	Body      string
	CreatedAt string
}

func (s *Store) ListNotes(ctx context.Context, personID int64) ([]Note, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, body, created_at FROM notes WHERE person_id = ? ORDER BY created_at DESC
	`, personID)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	var items []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Body, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		items = append(items, n)
	}
	return items, rows.Err()
}
