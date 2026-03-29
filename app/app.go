package app 

import (
	"os"
	"path/filepath"

	"github.com/tsusheel/kb-cli/db"
	"github.com/spf13/viper"
)

func InitApp() {
	basePath := viper.GetString("base_path")

	os.MkdirAll(basePath, 0755)

	dbPath := filepath.Join(basePath, "kb.db")

	db.InitDB(dbPath)
	db.RunMigrations()
}

