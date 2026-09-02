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

func TestCircleCreateUpdateDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	id, err := s.CreateCircle(ctx, "Weekend Climbers", "Saturday morning bouldering crew")
	if err != nil {
		t.Fatalf("CreateCircle: %v", err)
	}

	got, err := s.GetCircle(ctx, id)
	if err != nil || got == nil || got.Name != "Weekend Climbers" || got.Description != "Saturday morning bouldering crew" {
		t.Fatalf("unexpected circle after create: %+v, err=%v", got, err)
	}

	if err := s.UpdateCircle(ctx, Circle{ID: id, Name: "Climbing Crew", Description: "Now Sunday mornings"}); err != nil {
		t.Fatalf("UpdateCircle: %v", err)
	}
	got, err = s.GetCircle(ctx, id)
	if err != nil || got == nil || got.Name != "Climbing Crew" || got.Description != "Now Sunday mornings" {
		t.Fatalf("unexpected circle after update: %+v, err=%v", got, err)
	}

	if err := s.DeleteCircle(ctx, id); err != nil {
		t.Fatalf("DeleteCircle: %v", err)
	}
	got, err = s.GetCircle(ctx, id)
	if err != nil || got != nil {
		t.Fatalf("expected nil circle after delete, got %+v, err=%v", got, err)
	}
}

func TestCircleDetailAndMemberCount(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	circleID, err := s.CreateCircle(ctx, "Family", "")
	if err != nil {
		t.Fatalf("CreateCircle: %v", err)
	}
	p1, _ := s.CreatePerson(ctx, Person{FirstName: "Maya", LastName: "Chen"})
	p2, _ := s.CreatePerson(ctx, Person{FirstName: "Alex"})

	if err := s.AddPersonToCircle(ctx, p1, circleID, "Sister"); err != nil {
		t.Fatalf("AddPersonToCircle: %v", err)
	}
	if err := s.AddPersonToCircle(ctx, p2, circleID, ""); err != nil {
		t.Fatalf("AddPersonToCircle: %v", err)
	}

	circles, err := s.ListCircles(ctx)
	if err != nil || len(circles) != 1 || circles[0].MemberCount != 2 {
		t.Fatalf("expected 1 circle with MemberCount=2, got %+v, err=%v", circles, err)
	}

	detail, err := s.GetCircleDetail(ctx, circleID)
	if err != nil || detail == nil || len(detail.Members) != 2 {
		t.Fatalf("unexpected circle detail: %+v, err=%v", detail, err)
	}

	// Deleting the circle should cascade-remove memberships (surfaced via
	// each person's own membership list going empty), not just the circle
	// row itself.
	if err := s.DeleteCircle(ctx, circleID); err != nil {
		t.Fatalf("DeleteCircle: %v", err)
	}
	memberships, err := s.ListCircleMemberships(ctx, p1)
	if err != nil || len(memberships) != 0 {
		t.Fatalf("expected no memberships left for p1 after circle delete, got %+v, err=%v", memberships, err)
	}
}
