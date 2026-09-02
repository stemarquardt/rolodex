package importer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rolodex/internal/db"
	"rolodex/internal/model"
)

// fixture covers: a full-featured contact (name, phones with types, email,
// address, no-year birthday via "--MM-DD", a real Google label plus the
// "myContacts" system pseudo-label, note, org/title), a minimal FN-only
// contact, and a contact with Google's 1604 placeholder-year birthday. All
// three carry CATEGORIES so they're all "labeled" under the default import
// scope — see fixtureUnlabeled below for that behavior specifically.
const fixture = `BEGIN:VCARD
VERSION:3.0
N:Chen;Maya;;;
FN:Maya Chen
TEL;TYPE=CELL:415-555-0182
TEL;TYPE=HOME:415-555-0199
EMAIL;TYPE=HOME:maya@example.com
ADR;TYPE=HOME:;;123 Bouldering Way;Portland;OR;97201;USA
BDAY:--06-12
CATEGORIES:Rock Climbing Friends,myContacts
NOTE:Met at the bouldering wall
ORG:Acme Corp
TITLE:Engineer
END:VCARD
BEGIN:VCARD
VERSION:3.0
FN:Company Only Entry
CATEGORIES:myContacts
END:VCARD
BEGIN:VCARD
VERSION:3.0
N:Rivera;Alex;;;
FN:Alex Rivera
BDAY:1604-03-04
CATEGORIES:starred,myContacts
END:VCARD
`

// fixtureMixedLabels covers the labeled-vs-unlabeled split Takeout actually
// produces: a saved/labeled contact alongside an auto-collected one with no
// CATEGORIES field at all (e.g. a one-off email sender Google picked up on
// its own — never something the user organized).
const fixtureMixedLabels = `BEGIN:VCARD
VERSION:3.0
N:Chen;Maya;;;
FN:Maya Chen
CATEGORIES:myContacts
END:VCARD
BEGIN:VCARD
VERSION:3.0
N:Support;ADT;;;
FN:ADT Support
EMAIL:noreply@adt.example.com
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

	sum, err := Import(ctx, store, strings.NewReader(fixture), Options{})
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
	if sum.CirclesLinked != 1 { // "Rock Climbing Friends" only; myContacts/starred skipped
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

func TestImportSkipsUnlabeledByDefault(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	sum, err := Import(ctx, store, strings.NewReader(fixtureMixedLabels), Options{})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if sum.PeopleCreated != 1 || sum.CardsSkippedUnlabeled != 1 {
		t.Fatalf("expected 1 person created and 1 skipped as unlabeled, got %+v", sum)
	}
	if adt, err := store.FindPersonByName(ctx, "ADT", "Support"); err != nil || adt != nil {
		t.Fatalf("expected unlabeled ADT Support to be skipped, got %+v, err=%v", adt, err)
	}
}

func TestImportAllContactsIncludesUnlabeled(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	sum, err := Import(ctx, store, strings.NewReader(fixtureMixedLabels), Options{AllContacts: true})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if sum.PeopleCreated != 2 || sum.CardsSkippedUnlabeled != 0 {
		t.Fatalf("expected both contacts created with AllContacts, got %+v", sum)
	}
}

func TestImportIdempotentOnRerun(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	if _, err := Import(ctx, store, strings.NewReader(fixture), Options{}); err != nil {
		t.Fatalf("first Import: %v", err)
	}

	sum, err := Import(ctx, store, strings.NewReader(fixture), Options{})
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

	sum, err := Import(ctx, store, strings.NewReader(fixture), Options{DryRun: true})
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

func TestResolveSource(t *testing.T) {
	dir := t.TempDir()

	// A plain .vcf file resolves to itself.
	vcfPath := filepath.Join(dir, "contacts.vcf")
	if err := os.WriteFile(vcfPath, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write vcf: %v", err)
	}
	got, err := ResolveSource(vcfPath)
	if err != nil || got != vcfPath {
		t.Fatalf("expected file path unchanged, got %q, err=%v", got, err)
	}

	// A directory containing "All Contacts/All Contacts.vcf" resolves to
	// that file (the Takeout export layout).
	takeoutDir := filepath.Join(dir, "Contacts")
	allContactsDir := filepath.Join(takeoutDir, "All Contacts")
	if err := os.MkdirAll(allContactsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	allContactsFile := filepath.Join(allContactsDir, "All Contacts.vcf")
	if err := os.WriteFile(allContactsFile, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write All Contacts.vcf: %v", err)
	}
	got, err = ResolveSource(takeoutDir)
	if err != nil || got != allContactsFile {
		t.Fatalf("expected resolution to %q, got %q, err=%v", allContactsFile, got, err)
	}

	// A directory missing that structure is a clear error, not a guess.
	emptyDir := filepath.Join(dir, "Empty")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, err := ResolveSource(emptyDir); err == nil {
		t.Fatalf("expected an error for a directory without All Contacts/All Contacts.vcf")
	}
}
