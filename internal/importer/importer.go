// Package importer maps a Google Takeout vCard export onto Rolodex's Store.
// It's a one-time bulk-import tool (see cmd/import), not an ongoing sync —
// PLANNING.md deliberately defers vCard/CardDAV import-sync as a persistent
// feature; this only ever runs when a human invokes it.
package importer

import (
	"context"
	"fmt"
	"io"
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
	PeopleCreated      int
	PeopleSkipped      int // already existed (exact first+last name match)
	CardsSkippedNoName int // no usable name to import at all
	ContactInfoCreated int
	DatesCreated       int
	CirclesLinked      int
	NotesCreated       int
	FactsCreated       int
}

// Import decodes every vCard in r and creates the corresponding People (and
// attached ContactInfo/ImportantDates/Circles/Notes/Facts) in store. If
// dryRun is true, no writes happen — Summary reflects what would have been
// created, including the FindPersonByName lookups used to decide skips.
func Import(ctx context.Context, store *model.Store, r io.Reader, dryRun bool) (Summary, error) {
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

		existing, err := store.FindPersonByName(ctx, first, last)
		if err != nil {
			return sum, fmt.Errorf("find person by name: %w", err)
		}
		if existing != nil {
			sum.PeopleSkipped++
			continue
		}

		sum.PeopleCreated++
		if dryRun {
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
// into Circle memberships. Categories starting with "*" are Google's own
// system labels (e.g. "* myContacts") and are skipped.
func importCircles(ctx context.Context, store *model.Store, card vcard.Card, personID int64, sum *Summary) error {
	for _, cat := range card.Categories() {
		cat = strings.TrimSpace(cat)
		if cat == "" || strings.HasPrefix(cat, "*") {
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

func nonEmpty(vals ...string) []string {
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}
