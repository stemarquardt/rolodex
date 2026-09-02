package model

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"rolodex/internal/db"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return NewStore(sqlDB)
}

func TestCreateAndListPeople(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	id, err := s.CreatePerson(ctx, Person{FirstName: "Maya", LastName: "Chen", Location: "San Francisco, CA"})
	if err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}

	people, err := s.ListPeople(ctx, "")
	if err != nil {
		t.Fatalf("ListPeople: %v", err)
	}
	if len(people) != 1 || people[0].ID != id || people[0].FullName() != "Maya Chen" {
		t.Fatalf("unexpected people list: %+v", people)
	}

	filtered, err := s.ListPeople(ctx, "chen")
	if err != nil {
		t.Fatalf("ListPeople filtered: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected search to match, got %d results", len(filtered))
	}

	none, err := s.ListPeople(ctx, "nobody")
	if err != nil {
		t.Fatalf("ListPeople no match: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("expected no matches, got %d", len(none))
	}
}

func TestFindPersonByName(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	id, err := s.CreatePerson(ctx, Person{FirstName: "Maya", LastName: "Chen"})
	if err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}

	got, err := s.FindPersonByName(ctx, "maya", "CHEN")
	if err != nil || got == nil || got.ID != id {
		t.Fatalf("expected case-insensitive match, got %+v, err=%v", got, err)
	}

	got, err = s.FindPersonByName(ctx, "Maya", "Rivera")
	if err != nil || got != nil {
		t.Fatalf("expected no match on last name alone, got %+v, err=%v", got, err)
	}

	got, err = s.FindPersonByName(ctx, "Nobody", "Chen")
	if err != nil || got != nil {
		t.Fatalf("expected no match on first name alone, got %+v, err=%v", got, err)
	}
}

func TestGetPersonDetailAggregatesRelatedData(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	mayaID, err := s.CreatePerson(ctx, Person{FirstName: "Maya", LastName: "Chen"})
	if err != nil {
		t.Fatalf("CreatePerson maya: %v", err)
	}
	sisterID, err := s.CreatePerson(ctx, Person{FirstName: "Jo", LastName: "Chen"})
	if err != nil {
		t.Fatalf("CreatePerson jo: %v", err)
	}

	rawDB := s.db
	if _, err := rawDB.ExecContext(ctx, `INSERT INTO contact_info (person_id, type, value) VALUES (?, 'mobile', '415-555-0182')`, mayaID); err != nil {
		t.Fatalf("insert contact info: %v", err)
	}
	if _, err := rawDB.ExecContext(ctx, `INSERT INTO important_dates (person_id, type, label, month, day) VALUES (?, 'birthday', 'Birthday', 8, 24)`, mayaID); err != nil {
		t.Fatalf("insert important date: %v", err)
	}
	if _, err := rawDB.ExecContext(ctx, `INSERT INTO pets (person_id, name, species) VALUES (?, 'Pip', 'Tabby cat')`, mayaID); err != nil {
		t.Fatalf("insert pet: %v", err)
	}
	if _, err := rawDB.ExecContext(ctx, `INSERT INTO facts (person_id, label, value) VALUES (?, 'Go-to order', 'Iced oat latte')`, mayaID); err != nil {
		t.Fatalf("insert fact: %v", err)
	}
	if _, err := rawDB.ExecContext(ctx, `INSERT INTO notes (person_id, body) VALUES (?, 'Sent photos from the coast')`, mayaID); err != nil {
		t.Fatalf("insert note: %v", err)
	}
	var siblingTypeID int64
	if err := rawDB.QueryRowContext(ctx, `SELECT id FROM relationship_types WHERE name = 'Sibling'`).Scan(&siblingTypeID); err != nil {
		t.Fatalf("lookup sibling type: %v", err)
	}
	if _, err := rawDB.ExecContext(ctx, `INSERT INTO relationships (person_id, related_person_id, relationship_type_id) VALUES (?, ?, ?)`, mayaID, sisterID, siblingTypeID); err != nil {
		t.Fatalf("insert relationship: %v", err)
	}
	var circleID int64
	res, err := rawDB.ExecContext(ctx, `INSERT INTO circles (name) VALUES ('Creative')`)
	if err != nil {
		t.Fatalf("insert circle: %v", err)
	}
	circleID, _ = res.LastInsertId()
	if _, err := rawDB.ExecContext(ctx, `INSERT INTO person_circles (person_id, circle_id, note) VALUES (?, ?, 'Met at a gallery opening')`, mayaID, circleID); err != nil {
		t.Fatalf("insert person_circle: %v", err)
	}

	detail, err := s.GetPersonDetail(ctx, mayaID)
	if err != nil {
		t.Fatalf("GetPersonDetail: %v", err)
	}
	if detail == nil {
		t.Fatal("expected detail, got nil")
	}
	if len(detail.ContactInfo) != 1 || detail.ContactInfo[0].Value != "415-555-0182" {
		t.Errorf("unexpected contact info: %+v", detail.ContactInfo)
	}
	if len(detail.ImportantDates) != 1 || detail.ImportantDates[0].Label != "Birthday" {
		t.Errorf("unexpected important dates: %+v", detail.ImportantDates)
	}
	if len(detail.Relationships) != 1 || detail.Relationships[0].RelatedName != "Jo Chen" || detail.Relationships[0].TypeName != "Sibling" {
		t.Errorf("unexpected relationships: %+v", detail.Relationships)
	}
	if len(detail.CircleMemberships) != 1 || detail.CircleMemberships[0].CircleName != "Creative" {
		t.Errorf("unexpected circle memberships: %+v", detail.CircleMemberships)
	}
	if len(detail.Pets) != 1 || detail.Pets[0].Name != "Pip" {
		t.Errorf("unexpected pets: %+v", detail.Pets)
	}
	if len(detail.Facts) != 1 || detail.Facts[0].Label != "Go-to order" {
		t.Errorf("unexpected facts: %+v", detail.Facts)
	}
	if len(detail.Notes) != 1 || detail.Notes[0].Body != "Sent photos from the coast" {
		t.Errorf("unexpected notes: %+v", detail.Notes)
	}

	missing, err := s.GetPersonDetail(ctx, 99999)
	if err != nil {
		t.Fatalf("GetPersonDetail missing: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil for missing person, got %+v", missing)
	}
}

func TestUpdateAndDeletePerson(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	id, err := s.CreatePerson(ctx, Person{FirstName: "Maya", LastName: "Chen"})
	if err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}

	err = s.UpdatePerson(ctx, Person{
		ID: id, FirstName: "Maya", LastName: "Chen-Lee", Nickname: "May", Location: "Oakland, CA", NudgeEnabled: false,
	})
	if err != nil {
		t.Fatalf("UpdatePerson: %v", err)
	}

	detail, err := s.GetPersonDetail(ctx, id)
	if err != nil {
		t.Fatalf("GetPersonDetail: %v", err)
	}
	if detail.LastName != "Chen-Lee" || detail.Nickname != "May" || detail.Location != "Oakland, CA" || detail.NudgeEnabled {
		t.Fatalf("update did not apply: %+v", detail.Person)
	}

	if err := s.UpdatePerson(ctx, Person{ID: 99999, FirstName: "Nobody"}); err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows updating missing person, got %v", err)
	}

	if _, err := s.db.ExecContext(ctx, `INSERT INTO pets (person_id, name) VALUES (?, 'Pip')`, id); err != nil {
		t.Fatalf("insert pet: %v", err)
	}

	if err := s.DeletePerson(ctx, id); err != nil {
		t.Fatalf("DeletePerson: %v", err)
	}

	gone, err := s.GetPersonDetail(ctx, id)
	if err != nil {
		t.Fatalf("GetPersonDetail after delete: %v", err)
	}
	if gone != nil {
		t.Fatalf("expected person to be gone, got %+v", gone)
	}

	var petCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pets WHERE person_id = ?`, id).Scan(&petCount); err != nil {
		t.Fatalf("count pets: %v", err)
	}
	if petCount != 0 {
		t.Fatalf("expected cascade delete to remove pets, found %d", petCount)
	}
}

func TestSetNudgeEnabled(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	id, err := s.CreatePerson(ctx, Person{FirstName: "Maya"})
	if err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}

	got, err := s.GetPerson(ctx, id)
	if err != nil || got == nil || got.NudgeEnabled {
		t.Fatalf("expected nudge_enabled to default to false, got %+v, err=%v", got, err)
	}

	if err := s.SetNudgeEnabled(ctx, id, true); err != nil {
		t.Fatalf("SetNudgeEnabled(true): %v", err)
	}
	got, err = s.GetPerson(ctx, id)
	if err != nil || got == nil || !got.NudgeEnabled {
		t.Fatalf("expected nudge_enabled=true, got %+v, err=%v", got, err)
	}

	if err := s.SetNudgeEnabled(ctx, id, false); err != nil {
		t.Fatalf("SetNudgeEnabled(false): %v", err)
	}
	got, err = s.GetPerson(ctx, id)
	if err != nil || got == nil || got.NudgeEnabled {
		t.Fatalf("expected nudge_enabled=false, got %+v, err=%v", got, err)
	}

	if err := s.SetNudgeEnabled(ctx, 99999, true); err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows for missing person, got %v", err)
	}
}
