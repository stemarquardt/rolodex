package model

import (
	"context"
	"fmt"
)

type Fact struct {
	ID    int64
	Label string
	Value string
}

func (s *Store) ListFacts(ctx context.Context, personID int64) ([]Fact, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, label, value FROM facts WHERE person_id = ? ORDER BY id
	`, personID)
	if err != nil {
		return nil, fmt.Errorf("list facts: %w", err)
	}
	defer rows.Close()

	var items []Fact
	for rows.Next() {
		var f Fact
		if err := rows.Scan(&f.ID, &f.Label, &f.Value); err != nil {
			return nil, fmt.Errorf("scan fact: %w", err)
		}
		items = append(items, f)
	}
	return items, rows.Err()
}
