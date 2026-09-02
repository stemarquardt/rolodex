package importer

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"rolodex/internal/db"
	"rolodex/internal/model"
)

// fixture covers: a full-featured contact (name, phones with types, email,
// address, no-year birthday via "--MM-DD", Google label, note, org/title), a
// minimal FN-only contact, and a contact with Google's 1604 placeholder-year
// birthday.
const fixture = `BEGIN:VCARD
VERSION:3.0
N:Chen;Maya;;;
FN:Maya Chen
TEL;TYPE=CELL:415-555-0182
TEL;TYPE=HOME:415-555-0199
EMAIL;TYPE=HOME:maya@example.com
ADR;TYPE=HOME:;;123 Bouldering Way;Portland;OR;97201;USA
BDAY:--06-12
CATEGORIES:Rock Climbing Friends,* myContacts
NOTE:Met at the bouldering wall
ORG:Acme Corp
TITLE:Engineer
END:VCARD
BEGIN:VCARD
VERSION:3.0
FN:Company Only Entry
END:VCARD
BEGIN:VCARD
VERSION:3.0
N:Rivera;Alex;;;
FN:Alex Rivera
BDAY:1604-03-04
END:VCARD
`

func newTestStore(t *testing.T) *model.Store {
	t.Helper()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return model.NewStore(sqlDB)
}

func TestImport(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	sum, err := Import(ctx, store, strings.NewReader(fixture), false)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	// Maya Chen (full-featured) + Company Only Entry (FN, no N, split on
	// first space into first/last) + Alex Rivera = 3 people created.
	if sum.PeopleCreated != 3 {
		t.Fatalf("expected 3 people created, got %+v", sum)
	}
	if sum.ContactInfoCreated != 4 { // 2 phones + 1 email + 1 address, for Maya only
		t.Fatalf("expected 4 contact info rows, got %+v", sum)
	}
	if sum.DatesCreated != 2 { // Maya's --06-12, Alex's 1604 placeholder
		t.Fatalf("expected 2 important dates, got %+v", sum)
	}
	if sum.CirclesLinked != 1 { // "Rock Climbing Friends" only; "* myContacts" skipped
		t.Fatalf("expected 1 circle link, got %+v", sum)
	}
	if sum.NotesCreated != 1 {
		t.Fatalf("expected 1 note, got %+v", sum)
	}
	if sum.FactsCreated != 2 { // Company + Title
		t.Fatalf("expected 2 facts, got %+v", sum)
	}

	maya, err := store.FindPersonByName(ctx, "Maya", "Chen")
	if err != nil || maya == nil {
		t.Fatalf("expected to find Maya Chen: %v, err=%v", maya, err)
	}
	if maya.Location != "Portland, OR" {
		t.Fatalf("expected Location %q, got %q", "Portland, OR", maya.Location)
	}

	detail, err := store.GetPersonDetail(ctx, maya.ID)
	if err != nil || detail == nil {
		t.Fatalf("GetPersonDetail: %+v, err=%v", detail, err)
	}
	if len(detail.ImportantDates) != 1 || detail.ImportantDates[0].Month != 6 || detail.ImportantDates[0].Day != 12 || detail.ImportantDates[0].Year.Valid {
		t.Fatalf("unexpected birthday for Maya: %+v", detail.ImportantDates)
	}

	var mobileFound bool
	for _, ci := range detail.ContactInfo {
		if ci.Type == "mobile" && ci.Value == "415-555-0182" {
			mobileFound = true
		}
	}
	if !mobileFound {
		t.Fatalf("expected a mobile contact info entry, got %+v", detail.ContactInfo)
	}

	alex, err := store.FindPersonByName(ctx, "Alex", "Rivera")
	if err != nil || alex == nil {
		t.Fatalf("expected to find Alex Rivera: %v, err=%v", alex, err)
	}
	alexDetail, err := store.GetPersonDetail(ctx, alex.ID)
	if err != nil || alexDetail == nil || len(alexDetail.ImportantDates) != 1 || alexDetail.ImportantDates[0].Year.Valid {
		t.Fatalf("expected Alex's 1604-placeholder birthday to have no year: %+v, err=%v", alexDetail, err)
	}
	if alexDetail.ImportantDates[0].Month != 3 || alexDetail.ImportantDates[0].Day != 4 {
		t.Fatalf("unexpected birthday for Alex: %+v", alexDetail.ImportantDates[0])
	}
}

func TestImportIdempotentOnRerun(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	if _, err := Import(ctx, store, strings.NewReader(fixture), false); err != nil {
		t.Fatalf("first Import: %v", err)
	}

	sum, err := Import(ctx, store, strings.NewReader(fixture), false)
	if err != nil {
		t.Fatalf("second Import: %v", err)
	}
	if sum.PeopleCreated != 0 || sum.PeopleSkipped != 3 {
		t.Fatalf("expected second run to skip all 3 people, got %+v", sum)
	}

	people, err := store.ListPeople(ctx, "")
	if err != nil {
		t.Fatalf("ListPeople: %v", err)
	}
	if len(people) != 3 {
		t.Fatalf("expected no duplicate people after re-run, got %d", len(people))
	}
}

func TestImportDryRunWritesNothing(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	sum, err := Import(ctx, store, strings.NewReader(fixture), true)
	if err != nil {
		t.Fatalf("Import (dry-run): %v", err)
	}
	if sum.PeopleCreated != 3 {
		t.Fatalf("expected dry-run to still report 3 would-be-created people, got %+v", sum)
	}

	people, err := store.ListPeople(ctx, "")
	if err != nil {
		t.Fatalf("ListPeople: %v", err)
	}
	if len(people) != 0 {
		t.Fatalf("expected dry-run to write nothing, got %d people", len(people))
	}
}
