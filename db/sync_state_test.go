package db

import (
	"testing"
	"time"

	"github.com/tsusheel/kb-cli/models"
)

func TestSyncState(t *testing.T) {
	setupTestDB(t)

	provider := "postgres"

	// 1. Initial state should be zero time
	lastSync, err := GetLastSyncedAt(provider)
	if err != nil {
		t.Fatalf("GetLastSyncedAt failed: %v", err)
	}
	if !lastSync.IsZero() {
		t.Errorf("expected zero lastSync, got %v", lastSync)
	}

	// 2. Set sync time
	now := time.Now().Truncate(time.Second)
	if err := SetLastSyncedAt(provider, now); err != nil {
		t.Fatalf("SetLastSyncedAt failed: %v", err)
	}

	// 3. Read back
	got, err := GetLastSyncedAt(provider)
	if err != nil {
		t.Fatalf("GetLastSyncedAt failed: %v", err)
	}
	if !got.Equal(now) {
		t.Errorf("GetLastSyncedAt = %v, expected %v", got, now)
	}

	// 4. Test unsynced counts
	n := &models.Note{
		ID:        "syncnote111122223333444455556666",
		Note:      "Test Note for Sync",
		Type:      models.DefaultNote,
		Status:    models.Active,
		CreatedAt: now.Add(1 * time.Minute),
		UpdatedAt: now.Add(1 * time.Minute),
	}
	if err := CreateNote(n); err != nil {
		t.Fatalf("CreateNote failed: %v", err)
	}

	notesCount, logsCount, err := GetUnsyncedCounts(now)
	if err != nil {
		t.Fatalf("GetUnsyncedCounts failed: %v", err)
	}
	if notesCount != 1 {
		t.Errorf("expected 1 unsynced note, got %d", notesCount)
	}
	if logsCount != 0 {
		t.Errorf("expected 0 unsynced logs, got %d", logsCount)
	}
}
