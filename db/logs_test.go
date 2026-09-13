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

func TestGetDailyLogsFilter(t *testing.T) {
	setupTestDB(t)

	// Create logs on different days
	yesterday := time.Now().AddDate(0, 0, -1)
	twoDaysAgo := time.Now().AddDate(0, 0, -2)

	l1, _ := CreateDailyLog("Log from 2 days ago")
	DB.Exec(`UPDATE daily_logs SET created_at = ? WHERE id = ?`, twoDaysAgo, l1.ID)

	l2, _ := CreateDailyLog("Log from yesterday")
	DB.Exec(`UPDATE daily_logs SET created_at = ? WHERE id = ?`, yesterday, l2.ID)

	_, _ = CreateDailyLog("Log from today")

	// 1. Query all-time
	allLogs, err := GetDailyLogsFilter(nil, nil, true, false)
	if err != nil {
		t.Fatalf("GetDailyLogsFilter all-time failed: %v", err)
	}
	if len(allLogs) != 3 {
		t.Errorf("expected 3 logs for all-time query, got %d", len(allLogs))
	}

	// 2. Query date range (twoDaysAgo to yesterday inclusive)
	rangeLogs, err := GetDailyLogsFilter(&twoDaysAgo, &yesterday, true, false)
	if err != nil {
		t.Fatalf("GetDailyLogsFilter range failed: %v", err)
	}
	if len(rangeLogs) != 2 {
		t.Errorf("expected 2 logs for range query, got %d", len(rangeLogs))
	}

	// 3. Query single date (yesterday)
	yesterdayLogs, err := GetDailyLogsFilter(&yesterday, &yesterday, true, false)
	if err != nil {
		t.Fatalf("GetDailyLogsFilter single date failed: %v", err)
	}
	if len(yesterdayLogs) != 1 || yesterdayLogs[0].ID != l2.ID {
		t.Errorf("expected 1 yesterday log matching l2, got %v", yesterdayLogs)
	}

	// 4. Query with from only (yesterday onwards)
	now := time.Now()
	fromLogs, err := GetDailyLogsFilter(&yesterday, &now, true, false)
	if err != nil {
		t.Fatalf("GetDailyLogsFilter from query failed: %v", err)
	}
	if len(fromLogs) != 2 {
		t.Errorf("expected 2 logs for from query, got %d", len(fromLogs))
	}
}
