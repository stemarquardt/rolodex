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

// ListRelationships returns every relationship touching this person, from
// either side of the stored row: relationships where they're person_id use
// the type's forward name, and relationships where they're related_person_id
// use the type's reverse name (with the other person shown as related).
// A relationship is stored once (see RelationshipType.NameReverse) but must
// display correctly from both people's profiles.
func (s *Store) ListRelationships(ctx context.Context, personID int64) ([]RelationshipView, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, rt.name, p2.id, TRIM(p2.first_name || ' ' || p2.last_name)
		FROM relationships r
		JOIN relationship_types rt ON rt.id = r.relationship_type_id
		JOIN people p2 ON p2.id = r.related_person_id
		WHERE r.person_id = ?

		UNION ALL

		SELECT r.id, rt.name_reverse, p1.id, TRIM(p1.first_name || ' ' || p1.last_name)
		FROM relationships r
		JOIN relationship_types rt ON rt.id = r.relationship_type_id
		JOIN people p1 ON p1.id = r.person_id
		WHERE r.related_person_id = ?

		ORDER BY 2
	`, personID, personID)
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

func (s *Store) ListRelationshipTypes(ctx context.Context) ([]RelationshipType, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, name_reverse FROM relationship_types ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list relationship types: %w", err)
	}
	defer rows.Close()

	var items []RelationshipType
	for rows.Next() {
		var rt RelationshipType
		if err := rows.Scan(&rt.ID, &rt.Name, &rt.NameReverse); err != nil {
			return nil, fmt.Errorf("scan relationship type: %w", err)
		}
		items = append(items, rt)
	}
	return items, rows.Err()
}

// CreateRelationship records that personID is the relationshipTypeID of
// relatedPersonID (e.g. personID is the "Parent" of relatedPersonID). The
// reverse direction is derived at read time via RelationshipType.NameReverse,
// not stored as a second row.
func (s *Store) CreateRelationship(ctx context.Context, personID, relatedPersonID, relationshipTypeID int64) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO relationships (person_id, related_person_id, relationship_type_id) VALUES (?, ?, ?)
	`, personID, relatedPersonID, relationshipTypeID)
	if err != nil {
		return 0, fmt.Errorf("create relationship: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) DeleteRelationship(ctx context.Context, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM relationships WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete relationship: %w", err)
	}
	return nil
}
