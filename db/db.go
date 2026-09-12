package db

import (
	"database/sql"
	_ "embed"
	"log"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

//go:embed schema.sql
var schemaSQL string

func InitDB(path string) {
	var err error
	DB, err = sql.Open("sqlite", path)
	if err != nil {
		log.Fatal(err)
	}

	DB.SetMaxOpenConns(1)

	// Performance + safety
	_, err = DB.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA foreign_keys = ON;
	`)

	// test connection
	if err := DB.Ping(); err != nil {
		log.Fatal(err)
	}
}

func InitSchema() error {
	_, err := DB.Exec(schemaSQL)
	return err
}

func RunMigrations() error {
	return InitSchema()
}

func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

