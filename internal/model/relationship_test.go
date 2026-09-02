package model

import (
	"context"
	"testing"
)

func TestRelationshipBidirectionalDisplay(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	mayaID, _ := s.CreatePerson(ctx, Person{FirstName: "Maya", LastName: "Chen"})
	joID, _ := s.CreatePerson(ctx, Person{FirstName: "Jo", LastName: "Chen"})

	types, err := s.ListRelationshipTypes(ctx)
	if err != nil {
		t.Fatalf("ListRelationshipTypes: %v", err)
	}
	var parentTypeID int64
	for _, rt := range types {
		if rt.Name == "Parent" {
			parentTypeID = rt.ID
		}
	}
	if parentTypeID == 0 {
		t.Fatalf("expected seeded 'Parent' relationship type, got %+v", types)
	}

	// Maya is the Parent of Jo — stored as a single directional row.
	relID, err := s.CreateRelationship(ctx, mayaID, joID, parentTypeID)
	if err != nil {
		t.Fatalf("CreateRelationship: %v", err)
	}

	mayaRels, err := s.ListRelationships(ctx, mayaID)
	if err != nil {
		t.Fatalf("ListRelationships maya: %v", err)
	}
	if len(mayaRels) != 1 || mayaRels[0].TypeName != "Parent" || mayaRels[0].RelatedName != "Jo Chen" {
		t.Fatalf("unexpected relationships for maya: %+v", mayaRels)
	}

	joRels, err := s.ListRelationships(ctx, joID)
	if err != nil {
		t.Fatalf("ListRelationships jo: %v", err)
	}
	if len(joRels) != 1 || joRels[0].TypeName != "Child" || joRels[0].RelatedName != "Maya Chen" {
		t.Fatalf("expected jo to see the reverse label 'Child' pointing at maya, got: %+v", joRels)
	}

	if err := s.DeleteRelationship(ctx, relID); err != nil {
		t.Fatalf("DeleteRelationship: %v", err)
	}
	mayaRels, _ = s.ListRelationships(ctx, mayaID)
	joRels, _ = s.ListRelationships(ctx, joID)
	if len(mayaRels) != 0 || len(joRels) != 0 {
		t.Fatalf("expected relationship gone from both sides after delete: maya=%+v jo=%+v", mayaRels, joRels)
	}
}

func TestCreateAndDeleteRelationshipType(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	id, err := s.CreateRelationshipType(ctx, "Coworker", "Coworker")
	if err != nil {
		t.Fatalf("CreateRelationshipType: %v", err)
	}

	types, err := s.ListRelationshipTypes(ctx)
	if err != nil {
		t.Fatalf("ListRelationshipTypes: %v", err)
	}
	var found bool
	for _, rt := range types {
		if rt.ID == id {
			found = true
			if rt.Name != "Coworker" || rt.NameReverse != "Coworker" {
				t.Fatalf("unexpected relationship type: %+v", rt)
			}
		}
	}
	if !found {
		t.Fatalf("expected new relationship type in list, got %+v", types)
	}

	if err := s.DeleteRelationshipType(ctx, id); err != nil {
		t.Fatalf("DeleteRelationshipType: %v", err)
	}
	types, _ = s.ListRelationshipTypes(ctx)
	for _, rt := range types {
		if rt.ID == id {
			t.Fatalf("expected relationship type to be gone after delete, still found: %+v", rt)
		}
	}
}

func TestDeleteRelationshipTypeCascadesToRelationships(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	typeID, err := s.CreateRelationshipType(ctx, "Coworker", "Coworker")
	if err != nil {
		t.Fatalf("CreateRelationshipType: %v", err)
	}
	mayaID, _ := s.CreatePerson(ctx, Person{FirstName: "Maya"})
	joID, _ := s.CreatePerson(ctx, Person{FirstName: "Jo"})
	if _, err := s.CreateRelationship(ctx, mayaID, joID, typeID); err != nil {
		t.Fatalf("CreateRelationship: %v", err)
	}

	// Unlike option_values, relationship_type_id is a real foreign key with
	// ON DELETE CASCADE — deleting the type must also remove the
	// relationship using it, not just the choice from future pickers.
	if err := s.DeleteRelationshipType(ctx, typeID); err != nil {
		t.Fatalf("DeleteRelationshipType: %v", err)
	}
	rels, err := s.ListRelationships(ctx, mayaID)
	if err != nil {
		t.Fatalf("ListRelationships: %v", err)
	}
	if len(rels) != 0 {
		t.Fatalf("expected relationship to be cascade-deleted with its type, got %+v", rels)
	}
}
