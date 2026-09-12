package db

import (
	"testing"
	"time"

	"github.com/tsusheel/kb-cli/models"
)

func TestDailyLogLifecycle(t *testing.T) {
	setupTestDB(t)

	// 1. Create Daily Log
	logEntry, err := CreateDailyLog("Investigated SQLite WAL mode checkpoints")
	if err != nil {
		t.Fatalf("CreateDailyLog failed: %v", err)
	}
	if logEntry.ID == "" {
		t.Fatal("expected log ID")
	}

	// 2. Query today's logs
	todayLogs, err := GetDailyLogsForDate(time.Now(), false)
	if err != nil {
		t.Fatalf("GetDailyLogsForDate failed: %v", err)
	}
	if len(todayLogs) != 1 || todayLogs[0].Content != "Investigated SQLite WAL mode checkpoints" {
		t.Fatalf("unexpected today logs: %v", todayLogs)
	}

	// 3. Promote Daily Log to Note non-interactively
	promotedNote, err := PromoteDailyLog(logEntry.ID[:7], models.Idea, models.Raw)
	if err != nil {
		t.Fatalf("PromoteDailyLog failed: %v", err)
	}
	if promotedNote.Note != logEntry.Content {
		t.Errorf("promoted note content mismatch: %s", promotedNote.Note)
	}
	if promotedNote.Type != models.Idea {
		t.Errorf("promoted note type mismatch: %v", promotedNote.Type)
	}
	if promotedNote.Status != models.Raw {
		t.Errorf("promoted note status mismatch: %v", promotedNote.Status)
	}

	// 4. Verify log is now marked with note_id
	updatedLog, err := GetDailyLog(logEntry.ID)
	if err != nil {
		t.Fatalf("GetDailyLog failed: %v", err)
	}
	if updatedLog.NoteID != promotedNote.ID {
		t.Errorf("expected log NoteID %s, got %s", promotedNote.ID, updatedLog.NoteID)
	}

	// 5. Query unpromoted logs should now be empty
	unpromoted, err := GetUnpromotedDailyLogs()
	if err != nil {
		t.Fatalf("GetUnpromotedDailyLogs failed: %v", err)
	}
	if len(unpromoted) != 0 {
		t.Errorf("expected 0 unpromoted logs, got %d", len(unpromoted))
	}

	// 6. Test soft-delete on daily log
	logEntry2, _ := CreateDailyLog("Temporary fleeting thought")
	if err := SoftDeleteDailyLog(logEntry2.ID[:7], "deleted by user"); err != nil {
		t.Fatalf("SoftDeleteDailyLog failed: %v", err)
	}

	activeLogs, _ := GetDailyLogsForDate(time.Now(), true)
	if len(activeLogs) != 1 { // Only the first promoted log should be active
		t.Errorf("expected 1 active log, got %d", len(activeLogs))
	}
}
