package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"rolodex/internal/db"
	"rolodex/internal/model"
	"rolodex/internal/web"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	dbPath := getenv("DB_PATH", "./data/people.db")
	staticDir := getenv("STATIC_DIR", "./static")
	addr := getenv("ADDR", ":8080")

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatalf("create db directory: %v", err)
	}

	sqlDB, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer sqlDB.Close()

	store := model.NewStore(sqlDB)
	handlers := web.NewHandlers(store)
	router := web.NewRouter(handlers, staticDir)

	log.Printf("listening on %s (db=%s)", addr, dbPath)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
