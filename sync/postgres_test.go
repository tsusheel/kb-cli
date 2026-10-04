package sync

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
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
		NotesPushed:     2,
		NotesPulled:     3,
		TagsPushed:      1,
		TagsPulled:      0,
		LinksPushed:     0,
		LinksPulled:     1,
		LogsPushed:      4,
		LogsPulled:      2,
		AuditLogsPushed: 5,
		AuditLogsPulled: 2,
		Duration:        250 * time.Millisecond,
	}

	if stats.TotalPushed() != 12 {
		t.Errorf("TotalPushed = %d, expected 12", stats.TotalPushed())
	}
	if stats.TotalPulled() != 8 {
		t.Errorf("TotalPulled = %d, expected 8", stats.TotalPulled())
	}
	if stats.Duration != 250*time.Millisecond {
		t.Errorf("Duration = %v, expected 250ms", stats.Duration)
	}
}

func TestBatchPush_EmptyNoOp(t *testing.T) {
	// Passing empty slices should immediately return nil without executing queries
	if err := batchPushNotes(nil, nil); err != nil {
		t.Errorf("expected nil for empty notes batch, got: %v", err)
	}
	if err := batchPushTags(nil, nil); err != nil {
		t.Errorf("expected nil for empty tags batch, got: %v", err)
	}
	if err := batchDeleteNoteTags(nil, nil); err != nil {
		t.Errorf("expected nil for empty delete note_tags, got: %v", err)
	}
	if err := batchPushNoteTags(nil, nil); err != nil {
		t.Errorf("expected nil for empty note_tags batch, got: %v", err)
	}
	if err := batchPushLinks(nil, nil); err != nil {
		t.Errorf("expected nil for empty links batch, got: %v", err)
	}
	if err := batchPushDailyLogs(nil, nil); err != nil {
		t.Errorf("expected nil for empty logs batch, got: %v", err)
	}
	if err := batchPushAuditLogs(nil, nil); err != nil {
		t.Errorf("expected nil for empty audit logs batch, got: %v", err)
	}
}

func TestLocalDeltaQueries(t *testing.T) {
	setupTestDB(t)

	now := time.Now().Truncate(time.Second)

	// Create note, tag, link, log
	note := &models.Note{
		ID:        "syncnotetest11112222333344445555",
		Note:      "Local Delta Note",
		Type:      models.DefaultNote,
		Status:    models.Active,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.CreateNote(note); err != nil {
		t.Fatalf("CreateNote failed: %v", err)
	}

	if err := db.AddTag(note.ID, "sync-tag"); err != nil {
		t.Fatalf("AddTag failed: %v", err)
	}

	// GetAllNotesSince(now - 1h) should find note
	past := now.Add(-1 * time.Hour)
	notes, err := db.GetAllNotesSince(past)
	if err != nil || len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d (err: %v)", len(notes), err)
	}

	// GetAllNotesSince(now + 1h) should find 0 notes (incremental delta!)
	future := now.Add(1 * time.Hour)
	notesFuture, err := db.GetAllNotesSince(future)
	if err != nil || len(notesFuture) != 0 {
		t.Fatalf("expected 0 notes in future query, got %d (err: %v)", len(notesFuture), err)
	}

	// Tags query incremental delta
	tagsFuture, err := db.GetAllTagsSince(future)
	if err != nil || len(tagsFuture) != 0 {
		t.Fatalf("expected 0 tags in future query, got %d (err: %v)", len(tagsFuture), err)
	}

	// Note tags for note IDs
	noteTags, err := db.GetNoteTagsForNotes([]string{note.ID})
	if err != nil || len(noteTags) != 1 {
		t.Fatalf("expected 1 note tag, got %d (err: %v)", len(noteTags), err)
	}
}
