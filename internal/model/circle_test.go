package model

import (
	"context"
	"testing"
)

func TestGetOrCreateCircleByNameAndMembership(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	personID, _ := s.CreatePerson(ctx, Person{FirstName: "Maya"})

	id1, err := s.GetOrCreateCircleByName(ctx, "Rock Climbing Friends")
	if err != nil {
		t.Fatalf("GetOrCreateCircleByName (create): %v", err)
	}

	// Case-insensitive re-lookup should return the same circle, not create a duplicate.
	id2, err := s.GetOrCreateCircleByName(ctx, "rock climbing friends")
	if err != nil {
		t.Fatalf("GetOrCreateCircleByName (lookup): %v", err)
	}
	if id1 != id2 {
		t.Fatalf("expected same circle id for case-insensitive match, got %d and %d", id1, id2)
	}

	circles, err := s.ListCircles(ctx)
	if err != nil || len(circles) != 1 {
		t.Fatalf("expected exactly 1 circle, got %+v, err=%v", circles, err)
	}

	if err := s.AddPersonToCircle(ctx, personID, id1, "Met at the bouldering wall"); err != nil {
		t.Fatalf("AddPersonToCircle: %v", err)
	}

	memberships, err := s.ListCircleMemberships(ctx, personID)
	if err != nil || len(memberships) != 1 || memberships[0].Note.String != "Met at the bouldering wall" {
		t.Fatalf("unexpected memberships: %+v, err=%v", memberships, err)
	}

	// Re-adding should update the note, not create a duplicate row.
	if err := s.AddPersonToCircle(ctx, personID, id1, "Updated note"); err != nil {
		t.Fatalf("AddPersonToCircle (update): %v", err)
	}
	memberships, err = s.ListCircleMemberships(ctx, personID)
	if err != nil || len(memberships) != 1 || memberships[0].Note.String != "Updated note" {
		t.Fatalf("expected note to update in place: %+v, err=%v", memberships, err)
	}

	if err := s.RemovePersonFromCircle(ctx, personID, id1); err != nil {
		t.Fatalf("RemovePersonFromCircle: %v", err)
	}
	memberships, err = s.ListCircleMemberships(ctx, personID)
	if err != nil || len(memberships) != 0 {
		t.Fatalf("expected no memberships after remove, got %+v, err=%v", memberships, err)
	}
}
