package db

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

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

func RunMigrations() error {
	schema, err := os.ReadFile("db/schema.sql")
	if err != nil {
		return err
	}

	_, err = DB.Exec(string(schema))
	return err
}

