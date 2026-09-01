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

func (s *Store) CreateFact(ctx context.Context, personID int64, label, value string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO facts (person_id, label, value) VALUES (?, ?, ?)
	`, personID, label, value)
	if err != nil {
		return 0, fmt.Errorf("create fact: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) DeleteFact(ctx context.Context, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM facts WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete fact: %w", err)
	}
	return nil
}
