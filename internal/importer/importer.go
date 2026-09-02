// Package importer maps a Google Takeout vCard export onto Rolodex's Store.
// It's a one-time bulk-import tool (see cmd/import), not an ongoing sync —
// PLANNING.md deliberately defers vCard/CardDAV import-sync as a persistent
// feature; this only ever runs when a human invokes it.
package importer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-vcard"

	"rolodex/internal/model"
)

// googlePlaceholderYear is the year Google Contacts writes into BDAY when
// only a month/day is known (its stand-in for vCard's "--MM-DD" no-year
// form). Treated the same as no year.
const googlePlaceholderYear = 1604

type Summary struct {
	PeopleCreated         int
	PeopleSkipped         int // already existed (exact first+last name match)
	CardsSkippedNoName    int // no usable name to import at all
	CardsSkippedUnlabeled int // no CATEGORIES field — not one of the contacts the user actually organized (see Options.AllContacts)
	ContactInfoCreated    int
	DatesCreated          int
	CirclesLinked         int
	NotesCreated          int
	FactsCreated          int
}

// Options controls Import's behavior.
type Options struct {
	// DryRun does every lookup Import would normally do (including the
	// FindPersonByName dedup check) but skips every write, so Summary
	// reflects what a real run would do.
	DryRun bool
	// AllContacts imports every card, including ones with no CATEGORIES
	// field at all. Google Takeout exports include every contact it has
	// ever auto-collected (email senders, one-off numbers, etc.) alongside
	// the ones the user actually saved/labeled — by default Import only
	// imports the latter (any card with at least one Google Contacts
	// Label), since that's what a curated personal CRM wants.
	AllContacts bool
}

// Import decodes every vCard in r and creates the corresponding People (and
// attached ContactInfo/ImportantDates/Circles/Notes/Facts) in store.
func Import(ctx context.Context, store *model.Store, r io.Reader, opts Options) (Summary, error) {
	var sum Summary
	dec := vcard.NewDecoder(r)

	for {
		card, err := dec.Decode()
		if err == io.EOF {
			break
		}
		if err != nil {
			return sum, fmt.Errorf("decode vcard: %w", err)
		}

		first, last := cardName(card)
		if first == "" && last == "" {
			sum.CardsSkippedNoName++
			continue
		}

		if !opts.AllContacts && !isLabeled(card) {
			sum.CardsSkippedUnlabeled++
			continue
		}

		existing, err := store.FindPersonByName(ctx, first, last)
		if err != nil {
			return sum, fmt.Errorf("find person by name: %w", err)
		}
		if existing != nil {
			sum.PeopleSkipped++
			continue
		}

		sum.PeopleCreated++
		if opts.DryRun {
			continue
		}

		personID, err := createPersonFromCard(ctx, store, card, first, last)
		if err != nil {
			return sum, err
		}
		if err := importContactInfo(ctx, store, card, personID, &sum); err != nil {
			return sum, err
		}
		if err := importBirthday(ctx, store, card, personID, &sum); err != nil {
			return sum, err
		}
		if err := importCircles(ctx, store, card, personID, &sum); err != nil {
			return sum, err
		}
		if err := importNote(ctx, store, card, personID, &sum); err != nil {
			return sum, err
		}
		if err := importFacts(ctx, store, card, personID, &sum); err != nil {
			return sum, err
		}
	}

	return sum, nil
}

// cardName extracts first/last name from the structured N field, falling
// back to splitting FN (some exported cards, e.g. company-only entries,
// only have a formatted name).
func cardName(card vcard.Card) (first, last string) {
	if n := card.Name(); n != nil && (n.GivenName != "" || n.FamilyName != "") {
		return strings.TrimSpace(n.GivenName), strings.TrimSpace(n.FamilyName)
	}
	fn := strings.TrimSpace(card.PreferredValue(vcard.FieldFormattedName))
	if fn == "" {
		return "", ""
	}
	parts := strings.SplitN(fn, " ", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

// isLabeled reports whether a card has a CATEGORIES field at all — i.e. the
// user assigned it to at least one Google Contacts Label (even just the
// implicit "myContacts" every saved contact gets). Note: card.Categories()
// splits the raw value on "," and returns []string{""} for an absent field,
// not an empty slice, so it can't be used directly for this check — check
// the raw field value instead.
func isLabeled(card vcard.Card) bool {
	return strings.TrimSpace(card.Value(vcard.FieldCategories)) != ""
}

// googleSystemCategories are pseudo-labels Google Contacts applies itself
// (not something the user created) — plain values in practice, not marked
// with any special prefix.
var googleSystemCategories = map[string]bool{
	"mycontacts": true,
	"starred":    true,
}

func createPersonFromCard(ctx context.Context, store *model.Store, card vcard.Card, first, last string) (int64, error) {
	p := model.Person{FirstName: first, LastName: last, NudgeEnabled: true}
	if addr := card.Address(); addr != nil {
		p.Location = strings.TrimSpace(strings.Join(nonEmpty(addr.Locality, addr.Region), ", "))
	}
	id, err := store.CreatePerson(ctx, p)
	if err != nil {
		return 0, fmt.Errorf("create person %s %s: %w", first, last, err)
	}
	return id, nil
}

func importContactInfo(ctx context.Context, store *model.Store, card vcard.Card, personID int64, sum *Summary) error {
	for _, f := range card[vcard.FieldTelephone] {
		if f.Value == "" {
			continue
		}
		if _, err := store.CreateContactInfo(ctx, personID, telephoneType(f), f.Value); err != nil {
			return fmt.Errorf("create phone contact info: %w", err)
		}
		sum.ContactInfoCreated++
	}
	for _, f := range card[vcard.FieldEmail] {
		if f.Value == "" {
			continue
		}
		if _, err := store.CreateContactInfo(ctx, personID, "email", f.Value); err != nil {
			return fmt.Errorf("create email contact info: %w", err)
		}
		sum.ContactInfoCreated++
	}
	if addr := card.Address(); addr != nil {
		formatted := strings.TrimSpace(strings.Join(nonEmpty(
			addr.StreetAddress, addr.ExtendedAddress, addr.Locality, addr.Region, addr.PostalCode, addr.Country,
		), ", "))
		if formatted != "" {
			if _, err := store.CreateContactInfo(ctx, personID, "address", formatted); err != nil {
				return fmt.Errorf("create address contact info: %w", err)
			}
			sum.ContactInfoCreated++
		}
	}
	return nil
}

func telephoneType(f *vcard.Field) string {
	switch {
	case f.Params.HasType(vcard.TypeCell):
		return "mobile"
	case f.Params.HasType(vcard.TypeHome):
		return "home"
	case f.Params.HasType(vcard.TypeWork):
		return "work"
	}
	if types := f.Params.Types(); len(types) > 0 {
		return types[0]
	}
	return "phone"
}

func importBirthday(ctx context.Context, store *model.Store, card vcard.Card, personID int64, sum *Summary) error {
	raw := strings.TrimSpace(card.Value(vcard.FieldBirthday))
	if raw == "" {
		return nil
	}
	month, day, year, ok := parseBirthday(raw)
	if !ok {
		return nil // unrecognized date form — skip rather than fail the whole import
	}
	if _, err := store.CreateImportantDate(ctx, personID, "birthday", "Birthday", month, day, year); err != nil {
		return fmt.Errorf("create birthday: %w", err)
	}
	sum.DatesCreated++
	return nil
}

// parseBirthday handles vCard's "--MM-DD" no-year form and full "YYYY-MM-DD"
// dates, treating Google's 1604 placeholder year the same as no year.
func parseBirthday(raw string) (month, day int, year *int, ok bool) {
	if strings.HasPrefix(raw, "--") && len(raw) == 7 {
		m, errM := strconv.Atoi(raw[2:4])
		d, errD := strconv.Atoi(raw[5:7])
		if errM != nil || errD != nil {
			return 0, 0, nil, false
		}
		return m, d, nil, true
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return 0, 0, nil, false
	}
	if t.Year() == googlePlaceholderYear {
		return int(t.Month()), t.Day(), nil, true
	}
	y := t.Year()
	return int(t.Month()), t.Day(), &y, true
}

// importCircles turns Google Contacts "Labels" (exported as CATEGORIES)
// into Circle memberships. Google's own system labels (e.g. "myContacts",
// applied to every saved contact; "starred") are skipped.
func importCircles(ctx context.Context, store *model.Store, card vcard.Card, personID int64, sum *Summary) error {
	for _, cat := range card.Categories() {
		cat = strings.TrimSpace(cat)
		if cat == "" || googleSystemCategories[strings.ToLower(cat)] {
			continue
		}
		circleID, err := store.GetOrCreateCircleByName(ctx, cat)
		if err != nil {
			return fmt.Errorf("get or create circle %q: %w", cat, err)
		}
		if err := store.AddPersonToCircle(ctx, personID, circleID, ""); err != nil {
			return fmt.Errorf("add person to circle %q: %w", cat, err)
		}
		sum.CirclesLinked++
	}
	return nil
}

func importNote(ctx context.Context, store *model.Store, card vcard.Card, personID int64, sum *Summary) error {
	note := strings.TrimSpace(card.Value(vcard.FieldNote))
	if note == "" {
		return nil
	}
	if _, err := store.CreateNote(ctx, personID, note); err != nil {
		return fmt.Errorf("create note: %w", err)
	}
	sum.NotesCreated++
	return nil
}

func importFacts(ctx context.Context, store *model.Store, card vcard.Card, personID int64, sum *Summary) error {
	facts := []struct{ label, value string }{
		{"Company", strings.TrimSpace(card.Value(vcard.FieldOrganization))},
		{"Title", strings.TrimSpace(card.Value(vcard.FieldTitle))},
	}
	for _, f := range facts {
		if f.value == "" {
			continue
		}
		if _, err := store.CreateFact(ctx, personID, f.label, f.value); err != nil {
			return fmt.Errorf("create fact %s: %w", f.label, err)
		}
		sum.FactsCreated++
	}
	return nil
}

// ResolveSource turns a user-supplied path into the actual .vcf file to
// import. path may be a .vcf file directly, or the extracted Takeout
// "Contacts" directory (or any directory containing an "All Contacts"
// subfolder) — Google Takeout splits a contacts export into one folder per
// Label plus an "All Contacts" folder that's the superset of everything
// else (every labeled card's CATEGORIES field lists all its labels), so
// that's the only file that needs importing.
func ResolveSource(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", path, err)
	}
	if !info.IsDir() {
		return path, nil
	}

	candidate := filepath.Join(path, "All Contacts", "All Contacts.vcf")
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	return "", fmt.Errorf("%s is a directory but doesn't contain All Contacts/All Contacts.vcf — point the import command at a specific .vcf file instead", path)
}

func nonEmpty(vals ...string) []string {
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}
