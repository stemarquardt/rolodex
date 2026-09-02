package model

import (
	"context"
	"testing"
)

func TestOptionValueCreateListDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	// A fresh DB is already seeded (internal/db/seed.go) — a real event_kind
	// category exists with the default 3 values.
	kinds, err := s.ListOptionValues(ctx, CategoryEventKind)
	if err != nil || len(kinds) != 3 {
		t.Fatalf("expected 3 seeded event_kind values, got %+v, err=%v", kinds, err)
	}

	id, err := s.CreateOptionValue(ctx, CategoryEventKind, "wedding")
	if err != nil {
		t.Fatalf("CreateOptionValue: %v", err)
	}
	kinds, err = s.ListOptionValues(ctx, CategoryEventKind)
	if err != nil || len(kinds) != 4 || kinds[3].ID != id || kinds[3].Value != "wedding" {
		t.Fatalf("expected wedding appended, got %+v, err=%v", kinds, err)
	}

	if err := s.DeleteOptionValue(ctx, id); err != nil {
		t.Fatalf("DeleteOptionValue: %v", err)
	}
	kinds, err = s.ListOptionValues(ctx, CategoryEventKind)
	if err != nil || len(kinds) != 3 {
		t.Fatalf("expected 3 values after delete, got %+v, err=%v", kinds, err)
	}
}

func TestOptionValueCaseInsensitiveDuplicateRejected(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	if _, err := s.CreateOptionValue(ctx, CategoryContactInfoType, "mobile"); err == nil {
		t.Fatalf("expected case-insensitive duplicate of seeded 'Mobile' to be rejected")
	}
}

func TestOptionValuesScopedByCategory(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	if _, err := s.CreateOptionValue(ctx, CategoryEventKind, "reunion"); err != nil {
		t.Fatalf("CreateOptionValue: %v", err)
	}

	statuses, err := s.ListOptionValues(ctx, CategoryEventStatus)
	if err != nil {
		t.Fatalf("ListOptionValues: %v", err)
	}
	for _, v := range statuses {
		if v.Value == "reunion" {
			t.Fatalf("event_kind value leaked into event_status listing: %+v", statuses)
		}
	}
}

func TestListAllOptionValues(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	all, err := s.ListAllOptionValues(ctx)
	if err != nil {
		t.Fatalf("ListAllOptionValues: %v", err)
	}
	wantCategories := []string{CategoryEventKind, CategoryEventStatus, CategoryContactInfoType, CategoryImportantDateType}
	for _, cat := range wantCategories {
		if len(all[cat]) == 0 {
			t.Fatalf("expected seeded values for category %q, got %+v", cat, all)
		}
	}
	if len(all[CategoryEventStatus]) != 5 || all[CategoryEventStatus][0].Value != "Idea" || all[CategoryEventStatus][4].Value != "Cancelled" {
		t.Fatalf("expected event_status seed order Idea..Cancelled, got %+v", all[CategoryEventStatus])
	}
}
