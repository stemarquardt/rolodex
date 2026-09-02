// Command import is a one-time bulk-import tool: it reads a Google Takeout
// vCard (.vcf) export and creates the corresponding People (plus contact
// info, birthdays, circles, notes, and facts) directly in the Rolodex
// SQLite file. It is not an ongoing sync — run it once to seed a baseline,
// or again later against a fresh export (safe to re-run: people that
// already exist by exact first+last name are skipped, not duplicated).
//
// Run this while the docker compose server is stopped, or point -db at the
// file before starting the container — SQLite only supports one writer, and
// internal/db.Open caps the connection pool at 1.
//
// Usage:
//
//	go run ./cmd/import -dry-run ~/Downloads/contacts.vcf
//	go run ./cmd/import ~/Downloads/contacts.vcf
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"rolodex/internal/db"
	"rolodex/internal/importer"
	"rolodex/internal/model"
)

func main() {
	dbPath := flag.String("db", "./data/people.db", "path to the Rolodex SQLite file")
	dryRun := flag.Bool("dry-run", false, "parse and report what would be imported without writing anything")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: import [-db path] [-dry-run] <contacts.vcf>")
		os.Exit(2)
	}
	vcfPath := flag.Arg(0)

	if err := os.MkdirAll(filepath.Dir(*dbPath), 0o755); err != nil {
		log.Fatalf("create db directory: %v", err)
	}
	sqlDB, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer sqlDB.Close()
	store := model.NewStore(sqlDB)

	f, err := os.Open(vcfPath)
	if err != nil {
		log.Fatalf("open vcf: %v", err)
	}
	defer f.Close()

	if *dryRun {
		log.Printf("dry run: no changes will be written to %s", *dbPath)
	}

	sum, err := importer.Import(context.Background(), store, f, *dryRun)
	if err != nil {
		log.Fatalf("import: %v", err)
	}

	fmt.Printf("People created:        %d\n", sum.PeopleCreated)
	fmt.Printf("People skipped:        %d (already existed)\n", sum.PeopleSkipped)
	fmt.Printf("Cards skipped:         %d (no usable name)\n", sum.CardsSkippedNoName)
	fmt.Printf("Contact info created:  %d\n", sum.ContactInfoCreated)
	fmt.Printf("Important dates:       %d\n", sum.DatesCreated)
	fmt.Printf("Circle memberships:    %d\n", sum.CirclesLinked)
	fmt.Printf("Notes created:         %d\n", sum.NotesCreated)
	fmt.Printf("Facts created:         %d\n", sum.FactsCreated)
}
