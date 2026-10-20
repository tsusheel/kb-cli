package db

import (
	"path/filepath"
	"testing"
)

func TestMigrationsLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_migrations.db")

	if err := InitDB(dbPath); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer CloseDB()

	// Initial migration run
	if err := InitSchema(); err != nil {
		t.Fatalf("InitSchema failed: %v", err)
	}

	migrations, err := GetAppliedMigrations()
	if err != nil {
		t.Fatalf("GetAppliedMigrations failed: %v", err)
	}
	if len(migrations) < 2 {
		t.Fatalf("expected at least 2 migrations applied, got %d", len(migrations))
	}
	if migrations[0].Version != 1 || migrations[1].Version != 2 {
		t.Errorf("unexpected migration versions: %+v", migrations)
	}

	// Idempotency: re-running migrations shouldn't fail
	if err := InitSchema(); err != nil {
		t.Fatalf("second InitSchema call failed: %v", err)
	}

	// Test Plugin Migration runner
	pluginSQL := `CREATE TABLE test_plugin_custom (id TEXT PRIMARY KEY, val TEXT);`
	if err := ApplyPluginMigration("test_plugin", 1, pluginSQL); err != nil {
		t.Fatalf("ApplyPluginMigration failed: %v", err)
	}

	// Re-applying same migration with same checksum should succeed
	if err := ApplyPluginMigration("test_plugin", 1, pluginSQL); err != nil {
		t.Fatalf("re-applying identical plugin migration failed: %v", err)
	}

	// Altered checksum for same version should fail
	badSQL := `CREATE TABLE test_plugin_custom (id TEXT PRIMARY KEY, val_altered TEXT);`
	if err := ApplyPluginMigration("test_plugin", 1, badSQL); err == nil {
		t.Fatalf("expected error applying differing checksum for applied version, got nil")
	}

	pMigrations, err := GetPluginMigrations("test_plugin")
	if err != nil {
		t.Fatalf("GetPluginMigrations failed: %v", err)
	}
	if len(pMigrations) != 1 {
		t.Fatalf("expected 1 plugin migration, got %d", len(pMigrations))
	}
}
