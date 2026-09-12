package sync

import (
	"path/filepath"
	"testing"

	"github.com/tsusheel/kb-cli/db"
)

func setupTestDB(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_sync.db")
	db.InitDB(dbPath)
	t.Cleanup(func() {
		db.CloseDB()
	})
	if err := db.InitSchema(); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}
}

func TestPostgresClient_Validation(t *testing.T) {
	_, err := NewPostgresClient("")
	if err == nil {
		t.Errorf("expected error for empty connection string, got nil")
	}

	client, err := NewPostgresClient("postgres://user:pass@localhost:5432/dbname?sslmode=disable")
	if err != nil {
		t.Fatalf("NewPostgresClient failed: %v", err)
	}
	defer client.Close()

	if client.ConnStr != "postgres://user:pass@localhost:5432/dbname?sslmode=disable" {
		t.Errorf("unexpected ConnStr: %s", client.ConnStr)
	}
}

func TestSyncStats(t *testing.T) {
	stats := &SyncStats{
		NotesPushed: 2,
		NotesPulled: 3,
		TagsPushed:  1,
		TagsPulled:  0,
		LinksPushed: 0,
		LinksPulled: 1,
		LogsPushed:  4,
		LogsPulled:  2,
	}

	if stats.TotalPushed() != 7 {
		t.Errorf("TotalPushed = %d, expected 7", stats.TotalPushed())
	}
	if stats.TotalPulled() != 6 {
		t.Errorf("TotalPulled = %d, expected 6", stats.TotalPulled())
	}
}
