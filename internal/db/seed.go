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

// seed inserts default lookup data on first run. Safe to call on every
// startup — it only inserts when the table is empty.
func seed(sqlDB *sql.DB) error {
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
