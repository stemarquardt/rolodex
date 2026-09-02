package db

import (
	"path/filepath"
	"testing"
)

func TestOpenAppliesSchemaAndSeeds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	sqlDB, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer sqlDB.Close()

	var count int
	if err := sqlDB.QueryRow("SELECT COUNT(*) FROM relationship_types").Scan(&count); err != nil {
		t.Fatalf("query relationship_types: %v", err)
	}
	if count != len(defaultRelationshipTypes) {
		t.Fatalf("expected %d seeded relationship types, got %d", len(defaultRelationshipTypes), count)
	}

	if _, err := sqlDB.Exec("INSERT INTO people (first_name, last_name) VALUES ('Ada', 'Lovelace')"); err != nil {
		t.Fatalf("insert person: %v", err)
	}

	// Re-opening (simulating a restart) must not error or duplicate seed data.
	sqlDB.Close()
	sqlDB2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer sqlDB2.Close()

	if err := sqlDB2.QueryRow("SELECT COUNT(*) FROM relationship_types").Scan(&count); err != nil {
		t.Fatalf("query relationship_types after reopen: %v", err)
	}
	if count != len(defaultRelationshipTypes) {
		t.Fatalf("expected seed to stay at %d after reopen, got %d", len(defaultRelationshipTypes), count)
	}
}

func TestOptionValuesSeedCapitalized(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	sqlDB, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer sqlDB.Close()

	var value string
	if err := sqlDB.QueryRow("SELECT value FROM option_values WHERE category = 'event_kind' AND value = 'Visit'").Scan(&value); err != nil {
		t.Fatalf("expected seeded event_kind value 'Visit', query failed: %v", err)
	}
}

func TestRenameLegacyOptionValuesMigratesOldLowercaseData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	sqlDB, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Simulate a database seeded before option values were capitalized, plus
	// a custom value a user added themselves — the migration must fix the
	// former and leave the latter untouched.
	if _, err := sqlDB.Exec("UPDATE option_values SET value = 'visit' WHERE category = 'event_kind' AND value = 'Visit'"); err != nil {
		t.Fatalf("simulate legacy lowercase value: %v", err)
	}
	if _, err := sqlDB.Exec("INSERT INTO option_values (category, value) VALUES ('event_kind', 'reunion')"); err != nil {
		t.Fatalf("insert custom value: %v", err)
	}
	sqlDB.Close()

	// Re-opening re-runs seed(), which includes renameLegacyOptionValues.
	sqlDB2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer sqlDB2.Close()

	var count int
	if err := sqlDB2.QueryRow("SELECT COUNT(*) FROM option_values WHERE category = 'event_kind' AND value = 'visit'").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected legacy lowercase 'visit' to be renamed, still found %d row(s)", count)
	}
	if err := sqlDB2.QueryRow("SELECT COUNT(*) FROM option_values WHERE category = 'event_kind' AND value = 'Visit'").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one 'Visit' after migration, got %d", count)
	}
	if err := sqlDB2.QueryRow("SELECT COUNT(*) FROM option_values WHERE category = 'event_kind' AND value = 'reunion'").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected custom value 'reunion' to survive the migration untouched, got %d", count)
	}
}
