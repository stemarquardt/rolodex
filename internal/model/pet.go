package model

import (
	"context"
	"fmt"
)

type Pet struct {
	ID      int64
	Name    string
	Species string
	Notes   string
}

func (s *Store) ListPets(ctx context.Context, personID int64) ([]Pet, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, species, notes FROM pets WHERE person_id = ? ORDER BY id
	`, personID)
	if err != nil {
		return nil, fmt.Errorf("list pets: %w", err)
	}
	defer rows.Close()

	var items []Pet
	for rows.Next() {
		var p Pet
		if err := rows.Scan(&p.ID, &p.Name, &p.Species, &p.Notes); err != nil {
			return nil, fmt.Errorf("scan pet: %w", err)
		}
		items = append(items, p)
	}
	return items, rows.Err()
}
