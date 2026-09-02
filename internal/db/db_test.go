package db

import (
	"database/sql"
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

func TestNudgeEnabledDefaultsToZeroForNewRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	sqlDB, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer sqlDB.Close()

	// Insert the way the model layer does — omitting nudge_enabled relies
	// on the schema default. On a fresh DB that default is 0 (opt-in).
	if _, err := sqlDB.Exec("INSERT INTO people (first_name) VALUES ('Ada')"); err != nil {
		t.Fatalf("insert person: %v", err)
	}
	var nudge int
	if err := sqlDB.QueryRow("SELECT nudge_enabled FROM people WHERE first_name = 'Ada'").Scan(&nudge); err != nil {
		t.Fatalf("query: %v", err)
	}
	if nudge != 0 {
		t.Fatalf("expected nudge_enabled to default to 0, got %d", nudge)
	}
}

func TestNudgeEnabledMigrationClearsExistingRowsExactlyOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	// Simulate a database from before the opt-in-by-default change (and
	// before schema_migrations existed at all): apply just the schema DDL
	// directly, without seed()/the migration, then insert two people
	// already enabled, as they would have been under the old opt-out
	// default. This is what an already-deployed database actually looks
	// like the moment it's opened by a binary that has this migration —
	// going through the normal Open() first would run (and immediately
	// mark applied) the migration against an empty people table, which
	// doesn't represent a real upgrade.
	sqlDB, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if _, err := sqlDB.Exec(schemaSQL); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	if _, err := sqlDB.Exec("INSERT INTO people (first_name, nudge_enabled) VALUES ('Ada', 1), ('Grace', 1)"); err != nil {
		t.Fatalf("insert people: %v", err)
	}
	sqlDB.Close()

	// Re-opening for real runs seed(), which includes the one-time
	// migration — both existing people should get cleared to match the new
	// default.
	sqlDB2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}

	var enabledCount int
	if err := sqlDB2.QueryRow("SELECT COUNT(*) FROM people WHERE nudge_enabled = 1").Scan(&enabledCount); err != nil {
		t.Fatalf("query: %v", err)
	}
	if enabledCount != 0 {
		t.Fatalf("expected migration to clear nudge_enabled for all existing people, %d still enabled", enabledCount)
	}

	// The critical part: the migration must never re-run. Manually opt
	// someone back in (as a user would via the UI), then reopen again — it
	// must survive, not get silently reset by a re-run matching on content.
	if _, err := sqlDB2.Exec("UPDATE people SET nudge_enabled = 1 WHERE first_name = 'Ada'"); err != nil {
		t.Fatalf("re-enable Ada: %v", err)
	}
	sqlDB2.Close()

	sqlDB3, err := Open(path)
	if err != nil {
		t.Fatalf("reopen again: %v", err)
	}
	defer sqlDB3.Close()

	var adaNudge int
	if err := sqlDB3.QueryRow("SELECT nudge_enabled FROM people WHERE first_name = 'Ada'").Scan(&adaNudge); err != nil {
		t.Fatalf("query: %v", err)
	}
	if adaNudge != 1 {
		t.Fatalf("expected Ada's manually-restored nudge_enabled=1 to survive a restart, got %d", adaNudge)
	}
}
