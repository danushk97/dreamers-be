package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/pressly/goose/v3"
	_ "github.com/lib/pq"

	"github.com/dreamers-be/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	dir := cfg.Database.MigrationPath
	if dir == "" {
		dir = "./migrations"
	}

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("set dialect: %v", err)
	}

	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	if err := goose.Run(cmd, db, dir); err != nil {
		log.Fatalf("goose %s: %v", cmd, err)
	}
	log.Printf("Migration %s completed successfully", cmd)
}
