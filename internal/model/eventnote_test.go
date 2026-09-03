package model

import (
	"context"
	"testing"
)

func TestEventNoteCreateAndDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	eventID, _ := s.CreateEvent(ctx, Event{Kind: "Visit", Status: "Tentative"})

	id, err := s.CreateEventNote(ctx, eventID, "Confirmed dates with everyone")
	if err != nil {
		t.Fatalf("CreateEventNote: %v", err)
	}
	notes, err := s.ListEventNotes(ctx, eventID)
	if err != nil || len(notes) != 1 || notes[0].ID != id || notes[0].Body != "Confirmed dates with everyone" {
		t.Fatalf("unexpected event notes: %+v, err=%v", notes, err)
	}

	if err := s.DeleteEventNote(ctx, id); err != nil {
		t.Fatalf("DeleteEventNote: %v", err)
	}
	notes, err = s.ListEventNotes(ctx, eventID)
	if err != nil || len(notes) != 0 {
		t.Fatalf("expected no event notes after delete, got %+v, err=%v", notes, err)
	}
}

func TestDeleteEventCascadesToEventNotes(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	eventID, _ := s.CreateEvent(ctx, Event{Kind: "Visit", Status: "Tentative"})

	if _, err := s.CreateEventNote(ctx, eventID, "Some note"); err != nil {
		t.Fatalf("CreateEventNote: %v", err)
	}

	if err := s.DeleteEvent(ctx, eventID); err != nil {
		t.Fatalf("DeleteEvent: %v", err)
	}

	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM event_notes WHERE event_id = ?`, eventID).Scan(&count); err != nil {
		t.Fatalf("count event_notes: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected event notes to be cascade-deleted with their event, found %d", count)
	}
}
