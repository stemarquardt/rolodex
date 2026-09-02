package model

import (
	"context"
	"testing"
)

func TestEventCreateUpdateDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	id, err := s.CreateEvent(ctx, Event{Kind: "visit", Status: "idea", Title: "Maybe a cabin weekend"})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}

	got, err := s.GetEvent(ctx, id)
	if err != nil || got == nil || got.Kind != "visit" || got.Status != "idea" || got.Title != "Maybe a cabin weekend" {
		t.Fatalf("unexpected event after create: %+v, err=%v", got, err)
	}

	got.Status = "confirmed"
	got.StartDate.String, got.StartDate.Valid = "2026-10-18", true
	if err := s.UpdateEvent(ctx, *got); err != nil {
		t.Fatalf("UpdateEvent: %v", err)
	}
	got, err = s.GetEvent(ctx, id)
	if err != nil || got == nil || got.Status != "confirmed" || !got.StartDate.Valid || got.StartDate.String != "2026-10-18" {
		t.Fatalf("unexpected event after update: %+v, err=%v", got, err)
	}

	if err := s.DeleteEvent(ctx, id); err != nil {
		t.Fatalf("DeleteEvent: %v", err)
	}
	got, err = s.GetEvent(ctx, id)
	if err != nil || got != nil {
		t.Fatalf("expected nil event after delete, got %+v, err=%v", got, err)
	}
}

func TestEventParticipantsAndDetail(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	eventID, err := s.CreateEvent(ctx, Event{Kind: "visit", Status: "tentative"})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	p1, _ := s.CreatePerson(ctx, Person{FirstName: "Maya", LastName: "Chen"})
	p2, _ := s.CreatePerson(ctx, Person{FirstName: "Alex"})

	if err := s.AddPersonToEvent(ctx, eventID, p1); err != nil {
		t.Fatalf("AddPersonToEvent: %v", err)
	}
	if err := s.AddPersonToEvent(ctx, eventID, p2); err != nil {
		t.Fatalf("AddPersonToEvent: %v", err)
	}

	// Re-adding the same person shouldn't error or duplicate.
	if err := s.AddPersonToEvent(ctx, eventID, p1); err != nil {
		t.Fatalf("AddPersonToEvent (re-add): %v", err)
	}

	detail, err := s.GetEventDetail(ctx, eventID)
	if err != nil || detail == nil || len(detail.People) != 2 {
		t.Fatalf("unexpected event detail: %+v, err=%v", detail, err)
	}

	if err := s.RemovePersonFromEvent(ctx, eventID, p2); err != nil {
		t.Fatalf("RemovePersonFromEvent: %v", err)
	}
	people, err := s.ListEventPeople(ctx, eventID)
	if err != nil || len(people) != 1 || people[0].ID != p1 {
		t.Fatalf("unexpected event people after remove: %+v, err=%v", people, err)
	}

	// Deleting the event should cascade-remove the remaining participant row.
	if err := s.DeleteEvent(ctx, eventID); err != nil {
		t.Fatalf("DeleteEvent: %v", err)
	}
	people, err = s.ListEventPeople(ctx, eventID)
	if err != nil || len(people) != 0 {
		t.Fatalf("expected no participants left after event delete, got %+v, err=%v", people, err)
	}
}

func TestListEventsGroupOrdering(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	doneID, _ := s.CreateEvent(ctx, Event{Kind: "visit", Status: "done"})
	confirmedID, _ := s.CreateEvent(ctx, Event{Kind: "visit", Status: "confirmed"})
	tentativeID, _ := s.CreateEvent(ctx, Event{Kind: "visit", Status: "tentative"})
	ideaID, _ := s.CreateEvent(ctx, Event{Kind: "visit", Status: "idea"})

	events, err := s.ListEvents(ctx)
	if err != nil || len(events) != 4 {
		t.Fatalf("ListEvents: %+v, err=%v", events, err)
	}

	wantOrder := []int64{ideaID, tentativeID, confirmedID, doneID}
	for i, id := range wantOrder {
		if events[i].ID != id {
			t.Fatalf("expected event order %v, got ids %v", wantOrder, eventIDs(events))
		}
	}
}

func eventIDs(events []Event) []int64 {
	ids := make([]int64, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}
	return ids
}
