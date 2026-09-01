package model

import (
	"context"
	"fmt"
)

type ContactInfo struct {
	ID    int64
	Type  string
	Value string
}

func (s *Store) ListContactInfo(ctx context.Context, personID int64) ([]ContactInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, value FROM contact_info WHERE person_id = ? ORDER BY id
	`, personID)
	if err != nil {
		return nil, fmt.Errorf("list contact info: %w", err)
	}
	defer rows.Close()

	var items []ContactInfo
	for rows.Next() {
		var c ContactInfo
		if err := rows.Scan(&c.ID, &c.Type, &c.Value); err != nil {
			return nil, fmt.Errorf("scan contact info: %w", err)
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func (s *Store) CreateContactInfo(ctx context.Context, personID int64, typ, value string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO contact_info (person_id, type, value) VALUES (?, ?, ?)
	`, personID, typ, value)
	if err != nil {
		return 0, fmt.Errorf("create contact info: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) DeleteContactInfo(ctx context.Context, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM contact_info WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete contact info: %w", err)
	}
	return nil
}
