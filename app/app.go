package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/tsusheel/kb-cli/db"
)

func InitApp() {
	basePath := viper.GetString("base_path")
	if basePath == "" {
		home, _ := os.UserHomeDir()
		basePath = filepath.Join(home, ".config", "kb")
	}

	_ = os.MkdirAll(basePath, 0755)

	dbPath := filepath.Join(basePath, "kb.db")

	db.InitDB(dbPath)
	if err := db.InitSchema(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database schema: %v\n", err)
		os.Exit(1)
	}
}
