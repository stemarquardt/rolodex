package model

import (
	"context"
	"database/sql"
	"testing"
)

func TestReminderCreateGetDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	personID, _ := s.CreatePerson(ctx, Person{FirstName: "Grandma", LastName: "Jean"})

	id, err := s.CreateReminder(ctx, Reminder{
		PersonID: sql.NullInt64{Int64: personID, Valid: true},
		DueDate:  "2026-09-10",
		Note:     "Call Grandma",
	})
	if err != nil {
		t.Fatalf("CreateReminder: %v", err)
	}

	got, err := s.GetReminder(ctx, id)
	if err != nil || got == nil || got.Note != "Call Grandma" || got.Status != "pending" || got.Recurring() {
		t.Fatalf("unexpected reminder after create: %+v, err=%v", got, err)
	}

	items, err := s.ListRemindersForPerson(ctx, personID)
	if err != nil || len(items) != 1 || items[0].PersonName != "Grandma Jean" {
		t.Fatalf("unexpected person reminders: %+v, err=%v", items, err)
	}

	if err := s.DeleteReminder(ctx, id); err != nil {
		t.Fatalf("DeleteReminder: %v", err)
	}
	items, err = s.ListRemindersForPerson(ctx, personID)
	if err != nil || len(items) != 0 {
		t.Fatalf("expected no reminders after delete, got %+v, err=%v", items, err)
	}
}

func TestCompleteReminderOneOff(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	id, err := s.CreateReminder(ctx, Reminder{DueDate: "2026-09-01", Note: "Send photos from the coast"})
	if err != nil {
		t.Fatalf("CreateReminder: %v", err)
	}

	if err := s.CompleteReminder(ctx, id); err != nil {
		t.Fatalf("CompleteReminder: %v", err)
	}

	got, err := s.GetReminder(ctx, id)
	if err != nil || got == nil || got.Status != "done" || got.DueDate != "2026-09-01" {
		t.Fatalf("expected one-off reminder marked done with unchanged due date, got %+v, err=%v", got, err)
	}
}

func TestCompleteReminderRecurring(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name     string
		unit     string
		interval int64
		wantDue  string
	}{
		{"days", "days", 3, "2026-09-04"},
		{"weeks", "weeks", 2, "2026-09-15"},
		{"months", "months", 1, "2026-10-01"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestStore(t)
			id, err := s.CreateReminder(ctx, Reminder{
				DueDate:            "2026-09-01",
				Note:               "Text every so often",
				RecurrenceInterval: sql.NullInt64{Int64: tc.interval, Valid: true},
				RecurrenceUnit:     sql.NullString{String: tc.unit, Valid: true},
			})
			if err != nil {
				t.Fatalf("CreateReminder: %v", err)
			}

			if err := s.CompleteReminder(ctx, id); err != nil {
				t.Fatalf("CompleteReminder: %v", err)
			}

			got, err := s.GetReminder(ctx, id)
			if err != nil || got == nil {
				t.Fatalf("GetReminder: %+v, err=%v", got, err)
			}
			if got.Status != "pending" {
				t.Fatalf("expected recurring reminder to stay pending, got status=%q", got.Status)
			}
			if got.DueDate != tc.wantDue {
				t.Fatalf("expected due_date advanced to %q, got %q", tc.wantDue, got.DueDate)
			}
		})
	}
}
