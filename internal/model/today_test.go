package model

import (
	"context"
	"testing"
	"time"
)

func TestListUpcomingImportantDates(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	personID, _ := s.CreatePerson(ctx, Person{FirstName: "Maya", LastName: "Chen"})

	soon := time.Now().AddDate(0, 0, 5)
	far := time.Now().AddDate(0, 0, 30)
	if _, err := s.CreateImportantDate(ctx, personID, "birthday", "Birthday", int(soon.Month()), soon.Day(), nil); err != nil {
		t.Fatalf("CreateImportantDate (soon): %v", err)
	}
	if _, err := s.CreateImportantDate(ctx, personID, "anniversary", "Anniversary", int(far.Month()), far.Day(), nil); err != nil {
		t.Fatalf("CreateImportantDate (far): %v", err)
	}

	upcoming, err := s.ListUpcomingImportantDates(ctx, 14)
	if err != nil {
		t.Fatalf("ListUpcomingImportantDates: %v", err)
	}
	if len(upcoming) != 1 {
		t.Fatalf("expected 1 upcoming date within 14 days, got %+v", upcoming)
	}
	if upcoming[0].Label != "Birthday" || upcoming[0].PersonID != personID || upcoming[0].DaysUntil != 5 {
		t.Fatalf("unexpected upcoming date: %+v", upcoming[0])
	}
}

func TestListEventsNeedingAttention(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	personID, _ := s.CreatePerson(ctx, Person{FirstName: "Sam"})

	soon := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	far := time.Now().AddDate(0, 0, 30).Format("2006-01-02")

	tentativeID := mustExec(t, s, `INSERT INTO events (kind, status, title) VALUES ('visit', 'tentative', 'Maybe visit')`)
	confirmedSoonID := mustExec(t, s, `INSERT INTO events (kind, status, title, start_date) VALUES ('visit', 'confirmed', 'Dinner', ?)`, soon)
	mustExec(t, s, `INSERT INTO events (kind, status, title, start_date) VALUES ('visit', 'confirmed', 'Distant trip', ?)`, far)
	mustExec(t, s, `INSERT INTO events (kind, status, title) VALUES ('visit', 'idea', 'Someday')`)
	mustExec(t, s, `INSERT INTO event_people (event_id, person_id) VALUES (?, ?)`, confirmedSoonID, personID)

	events, err := s.ListEventsNeedingAttention(ctx, 14)
	if err != nil {
		t.Fatalf("ListEventsNeedingAttention: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events needing attention, got %+v", events)
	}

	byID := map[int64]Event{}
	for _, e := range events {
		byID[e.ID] = e
	}
	if _, ok := byID[tentativeID]; !ok {
		t.Fatalf("expected tentative event to be included: %+v", events)
	}
	confirmed, ok := byID[confirmedSoonID]
	if !ok {
		t.Fatalf("expected soon-confirmed event to be included: %+v", events)
	}
	if len(confirmed.People) != 1 || confirmed.People[0].ID != personID {
		t.Fatalf("expected confirmed event to have participant attached: %+v", confirmed)
	}
}

func TestListEventsNeedingAttentionCaseInsensitive(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	soon := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	tentativeID := mustExec(t, s, `INSERT INTO events (kind, status, title) VALUES ('visit', 'Tentative', 'Maybe visit')`)
	confirmedID := mustExec(t, s, `INSERT INTO events (kind, status, title, start_date) VALUES ('visit', 'Confirmed', 'Dinner', ?)`, soon)

	events, err := s.ListEventsNeedingAttention(ctx, 14)
	if err != nil {
		t.Fatalf("ListEventsNeedingAttention: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected differently-cased Tentative/Confirmed events to still be included, got %+v", events)
	}
	byID := map[int64]bool{}
	for _, e := range events {
		byID[e.ID] = true
	}
	if !byID[tentativeID] || !byID[confirmedID] {
		t.Fatalf("expected both events by id, got %+v", events)
	}
}

func TestListDueReminders(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	personID, _ := s.CreatePerson(ctx, Person{FirstName: "Grandma"})

	today := time.Now().Format("2006-01-02")
	future := time.Now().AddDate(0, 0, 10).Format("2006-01-02")

	dueID := mustExec(t, s, `INSERT INTO reminders (person_id, due_date, note, status) VALUES (?, ?, 'Call Grandma', 'pending')`, personID, today)
	mustExec(t, s, `INSERT INTO reminders (due_date, note, status) VALUES (?, 'Not due yet', 'pending')`, future)
	mustExec(t, s, `INSERT INTO reminders (due_date, note, status) VALUES (?, 'Already done', 'done')`, today)

	reminders, err := s.ListDueReminders(ctx)
	if err != nil {
		t.Fatalf("ListDueReminders: %v", err)
	}
	if len(reminders) != 1 || reminders[0].ID != dueID {
		t.Fatalf("expected only the due pending reminder, got %+v", reminders)
	}
	if reminders[0].PersonName != "Grandma" {
		t.Fatalf("expected reminder to be annotated with person name, got %+v", reminders[0])
	}
	if reminders[0].Recurring() {
		t.Fatalf("expected non-recurring reminder, got %+v", reminders[0])
	}
}

func TestListStalePeople(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	neverContacted, _ := s.CreatePerson(ctx, Person{FirstName: "Never"})
	recentlyContacted, _ := s.CreatePerson(ctx, Person{FirstName: "Recent"})
	longQuiet, _ := s.CreatePerson(ctx, Person{FirstName: "Quiet"})
	nudgeOff, _ := s.CreatePerson(ctx, Person{FirstName: "Opted out"})
	if err := s.UpdatePerson(ctx, Person{ID: nudgeOff, FirstName: "Opted out", NudgeEnabled: false}); err != nil {
		t.Fatalf("UpdatePerson: %v", err)
	}

	if _, err := s.CreateNote(ctx, recentlyContacted, "just talked"); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	oldNoteID, err := s.CreateNote(ctx, longQuiet, "long ago")
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	oldTimestamp := time.Now().AddDate(0, 0, -90).Format("2006-01-02 15:04:05")
	mustExec(t, s, `UPDATE notes SET created_at = ? WHERE id = ?`, oldTimestamp, oldNoteID)
	mustExec(t, s, `INSERT INTO notes (person_id, body) VALUES (?, 'old note for opted-out person')`, nudgeOff)
	mustExec(t, s, `UPDATE notes SET created_at = ? WHERE person_id = ?`, oldTimestamp, nudgeOff)

	stale, err := s.ListStalePeople(ctx, 60)
	if err != nil {
		t.Fatalf("ListStalePeople: %v", err)
	}

	byID := map[int64]StalePerson{}
	for _, sp := range stale {
		byID[sp.PersonID] = sp
	}
	if len(stale) != 2 {
		t.Fatalf("expected 2 stale people, got %+v", stale)
	}
	if sp, ok := byID[neverContacted]; !ok || sp.DaysStale != -1 {
		t.Fatalf("expected never-contacted person with DaysStale=-1, got %+v (ok=%v)", byID[neverContacted], ok)
	}
	if sp, ok := byID[longQuiet]; !ok || sp.DaysStale < 60 {
		t.Fatalf("expected long-quiet person with DaysStale>=60, got %+v (ok=%v)", byID[longQuiet], ok)
	}
	if _, ok := byID[recentlyContacted]; ok {
		t.Fatalf("did not expect recently-contacted person in stale list: %+v", stale)
	}
	if _, ok := byID[nudgeOff]; ok {
		t.Fatalf("did not expect nudge-disabled person in stale list: %+v", stale)
	}
}

// mustExec runs a raw SQL statement against the store's underlying db —
// standing in for Event/Reminder creation, which don't have Store methods
// yet (those land with the Events/Reminders CRUD pages) — and returns the
// inserted row's id.
func mustExec(t *testing.T, s *Store, query string, args ...any) int64 {
	t.Helper()
	res, err := s.db.ExecContext(context.Background(), query, args...)
	if err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	return id
}
