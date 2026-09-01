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
