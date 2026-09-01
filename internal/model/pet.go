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

func (s *Store) CreatePet(ctx context.Context, personID int64, name, species, notes string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO pets (person_id, name, species, notes) VALUES (?, ?, ?, ?)
	`, personID, name, species, notes)
	if err != nil {
		return 0, fmt.Errorf("create pet: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) DeletePet(ctx context.Context, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM pets WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete pet: %w", err)
	}
	return nil
}
