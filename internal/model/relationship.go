package model

import (
	"context"
	"fmt"
)

type RelationshipType struct {
	ID          int64
	Name        string
	NameReverse string
}

// RelationshipView is a relationship joined with its type and the related
// person's name, ready to render on a profile page.
type RelationshipView struct {
	ID              int64
	TypeName        string
	RelatedPersonID int64
	RelatedName     string
}

func (s *Store) ListRelationships(ctx context.Context, personID int64) ([]RelationshipView, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, rt.name, p2.id, TRIM(p2.first_name || ' ' || p2.last_name)
		FROM relationships r
		JOIN relationship_types rt ON rt.id = r.relationship_type_id
		JOIN people p2 ON p2.id = r.related_person_id
		WHERE r.person_id = ?
		ORDER BY rt.name
	`, personID)
	if err != nil {
		return nil, fmt.Errorf("list relationships: %w", err)
	}
	defer rows.Close()

	var items []RelationshipView
	for rows.Next() {
		var r RelationshipView
		if err := rows.Scan(&r.ID, &r.TypeName, &r.RelatedPersonID, &r.RelatedName); err != nil {
			return nil, fmt.Errorf("scan relationship: %w", err)
		}
		items = append(items, r)
	}
	return items, rows.Err()
}
