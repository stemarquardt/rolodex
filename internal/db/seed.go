package db

import "database/sql"

type relationshipTypeSeed struct {
	name        string
	nameReverse string
}

var defaultRelationshipTypes = []relationshipTypeSeed{
	{"Parent", "Child"},
	{"Sibling", "Sibling"},
	{"Spouse/Partner", "Spouse/Partner"},
	{"Grandparent", "Grandchild"},
	{"Close Friend", "Close Friend"},
	{"Mentor", "Mentee"},
}

// defaultOptionValues seeds the managed suggestion lists behind the Event
// kind/status and ContactInfo/ImportantDate type <select> fields — see
// internal/model/optionvalue.go for the category constants these keys match.
var defaultOptionValues = map[string][]string{
	"event_kind":          {"Visit", "Trip", "Gathering"},
	"event_status":        {"Idea", "Tentative", "Confirmed", "Done", "Cancelled"},
	"contact_info_type":   {"Mobile", "Email", "Address", "Social"},
	"important_date_type": {"Birthday", "Anniversary", "Custom"},
}

// legacyOptionValueRenames maps the original (lowercase) seed values to
// their capitalized replacements above, so a database seeded before this
// change gets its default option_values rows renamed in place rather than
// staying stuck lowercase forever (seedOptionValues only runs against an
// empty table). Custom values a user has added themselves are untouched —
// this only ever renames an exact match against the old defaults.
var legacyOptionValueRenames = map[string]map[string]string{
	"event_kind":          {"visit": "Visit", "trip": "Trip", "gathering": "Gathering"},
	"event_status":        {"idea": "Idea", "tentative": "Tentative", "confirmed": "Confirmed", "done": "Done", "cancelled": "Cancelled"},
	"contact_info_type":   {"mobile": "Mobile", "email": "Email", "address": "Address", "social": "Social"},
	"important_date_type": {"birthday": "Birthday", "anniversary": "Anniversary", "custom": "Custom"},
}

// seed inserts default lookup data on first run, and applies small one-time
// renames/migrations to already-seeded data. Safe to call on every startup.
func seed(sqlDB *sql.DB) error {
	if err := seedRelationshipTypes(sqlDB); err != nil {
		return err
	}
	if err := seedOptionValues(sqlDB); err != nil {
		return err
	}
	if err := renameLegacyOptionValues(sqlDB); err != nil {
		return err
	}
	return applyOneTimeMigrations(sqlDB)
}

func seedRelationshipTypes(sqlDB *sql.DB) error {
	var count int
	if err := sqlDB.QueryRow("SELECT COUNT(*) FROM relationship_types").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, rt := range defaultRelationshipTypes {
		if _, err := tx.Exec(
			"INSERT INTO relationship_types (name, name_reverse) VALUES (?, ?)",
			rt.name, rt.nameReverse,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func seedOptionValues(sqlDB *sql.DB) error {
	var count int
	if err := sqlDB.QueryRow("SELECT COUNT(*) FROM option_values").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for category, values := range defaultOptionValues {
		for _, v := range values {
			if _, err := tx.Exec(
				"INSERT INTO option_values (category, value) VALUES (?, ?)",
				category, v,
			); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// renameLegacyOptionValues is idempotent: after the first successful run,
// none of the old lowercase values exist anymore, so later calls match
// nothing and do nothing.
func renameLegacyOptionValues(sqlDB *sql.DB) error {
	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for category, renames := range legacyOptionValueRenames {
		for oldValue, newValue := range renames {
			if _, err := tx.Exec(
				"UPDATE option_values SET value = ? WHERE category = ? AND value = ?",
				newValue, category, oldValue,
			); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// oneTimeMigrations are data changes that must run exactly once ever, not
// on every startup and not re-triggered by content matching — see
// schema_migrations in internal/db/schema.sql.
var oneTimeMigrations = []struct {
	name  string
	apply string
}{
	{
		name: "2026-09_nudge_enabled_opt_in_by_default",
		// Check-in nudges flipped from opt-out to opt-in — everyone
		// existing at the time gets cleared to match the new default
		// (false), rather than staying enabled from the old default. See
		// PLANNING.md for why: with a large bulk-imported contact list, an
		// opt-out default made "People going quiet" show almost everyone,
		// which defeats its purpose.
		apply: "UPDATE people SET nudge_enabled = 0",
	},
}

func applyOneTimeMigrations(sqlDB *sql.DB) error {
	for _, m := range oneTimeMigrations {
		var applied int
		if err := sqlDB.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE name = ?", m.name).Scan(&applied); err != nil {
			return err
		}
		if applied > 0 {
			continue
		}

		tx, err := sqlDB.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(m.apply); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.Exec("INSERT INTO schema_migrations (name) VALUES (?)", m.name); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
