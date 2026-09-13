package db

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tsusheel/kb-cli/models"
)

func TestAuditLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_audit.db")

	InitDB(dbPath)
	defer CloseDB()

	if err := InitSchema(); err != nil {
		t.Fatalf("InitSchema failed: %v", err)
	}

	// 1. Create a note and check audit entry
	noteID := "11112222-3333-4444-5555-666677778888"
	n := &models.Note{
		ID:        noteID,
		Note:      "Original Title",
		NoteFlesh: "Original Body",
		Type:      models.DefaultNote,
		Status:    models.Raw,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := CreateNote(n); err != nil {
		t.Fatalf("CreateNote failed: %v", err)
	}

	history, err := GetAuditHistory("note", noteID, 10)
	if err != nil {
		t.Fatalf("GetAuditHistory failed: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(history))
	}
	if history[0].Action != models.ActionCreated {
		t.Errorf("expected action 'created', got %s", history[0].Action)
	}

	// 2. Update the note and check diff summary
	n.Note = "Updated Title"
	n.Status = models.Active
	n.NoteFlesh = "Original Body with extra text"
	if err := UpdateNote(n); err != nil {
		t.Fatalf("UpdateNote failed: %v", err)
	}

	history, err = GetAuditHistory("note", noteID, 10)
	if err != nil {
		t.Fatalf("GetAuditHistory failed: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 audit entries, got %d", len(history))
	}
	if history[0].Action != models.ActionUpdated {
		t.Errorf("expected latest action 'updated', got %s", history[0].Action)
	}
	if !strings.Contains(history[0].ChangesSummary, "status: raw -> active") {
		t.Errorf("expected status diff in summary, got %s", history[0].ChangesSummary)
	}
	if !strings.Contains(history[0].ChangesSummary, "title") {
		t.Errorf("expected title diff in summary, got %s", history[0].ChangesSummary)
	}

	// 3. Revert to original snapshot
	origSnapshot := history[1].SnapshotJSON
	revertedNote, err := RevertNoteToSnapshot(noteID, origSnapshot)
	if err != nil {
		t.Fatalf("RevertNoteToSnapshot failed: %v", err)
	}
	if revertedNote.Note != "Original Title" || revertedNote.Status != models.Raw {
		t.Errorf("expected reverted values, got title=%q, status=%q", revertedNote.Note, revertedNote.Status)
	}

	// Check that a restored audit entry exists
	history, err = GetAuditHistory("note", noteID, 10)
	if err != nil {
		t.Fatalf("GetAuditHistory failed: %v", err)
	}
	if history[0].Action != models.ActionRestored {
		t.Errorf("expected action 'restored', got %s", history[0].Action)
	}

	// 4. Soft-delete and check audit entry
	if err := SoftDeleteNote(noteID, "no longer needed"); err != nil {
		t.Fatalf("SoftDeleteNote failed: %v", err)
	}

	history, err = GetAuditHistory("note", noteID, 10)
	if err != nil {
		t.Fatalf("GetAuditHistory failed: %v", err)
	}
	if history[0].Action != models.ActionDeleted {
		t.Errorf("expected action 'deleted', got %s", history[0].Action)
	}

	// 5. Test Global Recent Audits
	recent, err := GetRecentAuditHistory(10)
	if err != nil {
		t.Fatalf("GetRecentAuditHistory failed: %v", err)
	}
	if len(recent) < 4 {
		t.Errorf("expected at least 4 recent audit entries, got %d", len(recent))
	}
}

func TestDailyLogAudit(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_log_audit.db")

	InitDB(dbPath)
	defer CloseDB()

	if err := InitSchema(); err != nil {
		t.Fatalf("InitSchema failed: %v", err)
	}

	// 1. Create Daily Log
	logEntry, err := CreateDailyLog("Working on audit history feature")
	if err != nil {
		t.Fatalf("CreateDailyLog failed: %v", err)
	}

	history, err := GetAuditHistory("daily_log", logEntry.ID, 10)
	if err != nil {
		t.Fatalf("GetAuditHistory failed: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 audit entry for log, got %d", len(history))
	}
	if history[0].Action != models.ActionCreated {
		t.Errorf("expected action 'created', got %s", history[0].Action)
	}

	// 2. Promote Log
	note, err := PromoteDailyLog(logEntry.ID, models.Todo, models.Active)
	if err != nil {
		t.Fatalf("PromoteDailyLog failed: %v", err)
	}
	if note == nil {
		t.Fatalf("expected note to be created from log")
	}

	history, err = GetAuditHistory("daily_log", logEntry.ID, 10)
	if err != nil {
		t.Fatalf("GetAuditHistory failed: %v", err)
	}
	if history[0].Action != models.ActionPromoted {
		t.Errorf("expected action 'promoted', got %s", history[0].Action)
	}
}
