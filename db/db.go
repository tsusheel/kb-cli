package db

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

//go:embed schema.sql
var schemaSQL string

func InitDB(path string) error {
	if DB != nil {
		_ = DB.Close()
	}
	var err error
	DB, err = sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("failed opening sqlite database: %w", err)
	}

	DB.SetMaxOpenConns(1)

	// Performance + safety
	_, err = DB.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA foreign_keys = ON;
	`)
	if err != nil {
		return fmt.Errorf("failed executing pragmas: %w", err)
	}

	// test connection
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed pinging database: %w", err)
	}
	return nil
}

func InitSchema() error {
	_, err := DB.Exec(schemaSQL)
	return err
}

func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

