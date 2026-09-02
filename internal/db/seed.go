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
	"event_kind":          {"visit", "trip", "gathering"},
	"event_status":        {"idea", "tentative", "confirmed", "done", "cancelled"},
	"contact_info_type":   {"mobile", "email", "address", "social"},
	"important_date_type": {"birthday", "anniversary", "custom"},
}

// seed inserts default lookup data on first run. Safe to call on every
// startup — it only inserts when a given table is empty.
func seed(sqlDB *sql.DB) error {
	if err := seedRelationshipTypes(sqlDB); err != nil {
		return err
	}
	return seedOptionValues(sqlDB)
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
