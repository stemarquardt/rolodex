package model

import (
	"context"
	"testing"
)

func TestContactInfoCreateAndDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	personID, _ := s.CreatePerson(ctx, Person{FirstName: "Maya"})

	id, err := s.CreateContactInfo(ctx, personID, "mobile", "415-555-0182")
	if err != nil {
		t.Fatalf("CreateContactInfo: %v", err)
	}

	items, err := s.ListContactInfo(ctx, personID)
	if err != nil || len(items) != 1 || items[0].ID != id || items[0].Value != "415-555-0182" {
		t.Fatalf("unexpected contact info after create: %+v, err=%v", items, err)
	}

	if err := s.DeleteContactInfo(ctx, id); err != nil {
		t.Fatalf("DeleteContactInfo: %v", err)
	}
	items, err = s.ListContactInfo(ctx, personID)
	if err != nil || len(items) != 0 {
		t.Fatalf("expected empty contact info after delete, got %+v, err=%v", items, err)
	}
}

func TestImportantDateCreateAndDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	personID, _ := s.CreatePerson(ctx, Person{FirstName: "Maya"})

	id, err := s.CreateImportantDate(ctx, personID, "birthday", "Birthday", 8, 24, nil)
	if err != nil {
		t.Fatalf("CreateImportantDate: %v", err)
	}
	dates, err := s.ListImportantDates(ctx, personID)
	if err != nil || len(dates) != 1 || dates[0].Year.Valid {
		t.Fatalf("unexpected important dates: %+v, err=%v", dates, err)
	}

	year := 2020
	if _, err := s.CreateImportantDate(ctx, personID, "custom", "Big presentation", 6, 12, &year); err != nil {
		t.Fatalf("CreateImportantDate with year: %v", err)
	}
	dates, err = s.ListImportantDates(ctx, personID)
	if err != nil || len(dates) != 2 {
		t.Fatalf("expected 2 important dates, got %+v, err=%v", dates, err)
	}

	if err := s.DeleteImportantDate(ctx, id); err != nil {
		t.Fatalf("DeleteImportantDate: %v", err)
	}
	dates, err = s.ListImportantDates(ctx, personID)
	if err != nil || len(dates) != 1 {
		t.Fatalf("expected 1 important date after delete, got %+v, err=%v", dates, err)
	}
}

func TestPetCreateAndDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	personID, _ := s.CreatePerson(ctx, Person{FirstName: "Maya"})

	id, err := s.CreatePet(ctx, personID, "Pip", "Tabby cat", "loves string")
	if err != nil {
		t.Fatalf("CreatePet: %v", err)
	}
	pets, err := s.ListPets(ctx, personID)
	if err != nil || len(pets) != 1 || pets[0].ID != id {
		t.Fatalf("unexpected pets: %+v, err=%v", pets, err)
	}

	if err := s.DeletePet(ctx, id); err != nil {
		t.Fatalf("DeletePet: %v", err)
	}
	pets, err = s.ListPets(ctx, personID)
	if err != nil || len(pets) != 0 {
		t.Fatalf("expected no pets after delete, got %+v, err=%v", pets, err)
	}
}

func TestFactCreateAndDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	personID, _ := s.CreatePerson(ctx, Person{FirstName: "Maya"})

	id, err := s.CreateFact(ctx, personID, "Go-to order", "Iced oat latte")
	if err != nil {
		t.Fatalf("CreateFact: %v", err)
	}
	facts, err := s.ListFacts(ctx, personID)
	if err != nil || len(facts) != 1 || facts[0].ID != id {
		t.Fatalf("unexpected facts: %+v, err=%v", facts, err)
	}

	if err := s.DeleteFact(ctx, id); err != nil {
		t.Fatalf("DeleteFact: %v", err)
	}
	facts, err = s.ListFacts(ctx, personID)
	if err != nil || len(facts) != 0 {
		t.Fatalf("expected no facts after delete, got %+v, err=%v", facts, err)
	}
}

func TestNoteCreateAndDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	personID, _ := s.CreatePerson(ctx, Person{FirstName: "Maya"})

	id, err := s.CreateNote(ctx, personID, "Sent photos from the coast")
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	notes, err := s.ListNotes(ctx, personID)
	if err != nil || len(notes) != 1 || notes[0].ID != id {
		t.Fatalf("unexpected notes: %+v, err=%v", notes, err)
	}

	if err := s.DeleteNote(ctx, id); err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}
	notes, err = s.ListNotes(ctx, personID)
	if err != nil || len(notes) != 0 {
		t.Fatalf("expected no notes after delete, got %+v, err=%v", notes, err)
	}
}
