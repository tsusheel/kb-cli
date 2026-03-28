package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/tsusheel/kb-cli/internal/db"
)

func initApp() {
	home, _ := os.UserHomeDir()
	basePath := filepath.Join(home, ".kb-cli")

	os.MkdirAll(basePath, 0755)

	dbPath := viper.GetString("db_path")

	if dbPath == "" {
		dbPath = filepath.Join(home, ".kb-cli", "kb.db")
	}

	db.InitDB(dbPath)
	db.CreateTables()
}

