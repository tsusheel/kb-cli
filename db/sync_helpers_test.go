package db

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tsusheel/kb-cli/models"
)

func cleanUUID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

func TestApplyRemotePulls_AtomicBatch(t *testing.T) {
	setupTestDB(t)

	now := time.Now().Truncate(time.Second)
	noteID1 := cleanUUID()
	noteID2 := cleanUUID()
	tagID1 := "remote-tag-1"
	tagID2 := "remote-tag-2"
	linkID := cleanUUID()
	logID := cleanUUID()
	auditID := cleanUUID()

	pullData := &RemotePullData{
		Tags: []models.Tag{
			{ID: tagID1, Name: "batch-tag-1", CreatedAt: now},
			{ID: tagID2, Name: "batch-tag-2", CreatedAt: now},
		},
		Notes: []models.Note{
			{
				ID:             noteID1,
				Note:           "Pulled Note 1 for Batch Sync",
				NoteFlesh:      "Extended batch details",
				Type:           models.DefaultNote,
				Status:         models.Active,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			{
				ID:             noteID2,
				Note:           "Pulled Note 2 for Batch Sync",
				Type:           models.Idea,
				Status:         models.Active,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
		},
		ClearedNotes: []string{noteID1},
		NoteTags: []models.NoteTag{
			{NoteID: noteID1, TagID: tagID1, CreatedAt: now},
			{NoteID: noteID1, TagID: tagID2, CreatedAt: now},
		},
		Links: []models.Link{
			{
				ID:        linkID,
				FromNote:  noteID1,
				ToNote:    noteID2,
				Type:      models.RelatedTo,
				CreatedAt: now,
			},
		},
		DailyLogs: []models.DailyLog{
			{
				ID:        logID,
				Content:   "Batch synced daily log",
				CreatedAt: now,
			},
		},
		AuditLogs: []models.AuditEntry{
			{
				ID:             auditID,
				EntityType:     "note",
				EntityID:       noteID1,
				Action:         models.ActionCreated,
				ChangesSummary: "Created via batch sync",
				CreatedAt:      now,
			},
		},
	}

	if err := ApplyRemotePulls(pullData); err != nil {
		t.Fatalf("ApplyRemotePulls failed: %v", err)
	}

	// 1. Verify Note 1 exists and is searchable via FTS
	n1, err := GetNote(noteID1)
	if err != nil {
		t.Fatalf("GetNote(%s) failed: %v", noteID1, err)
	}
	if n1.Note != "Pulled Note 1 for Batch Sync" {
		t.Errorf("unexpected note content: %s", n1.Note)
	}

	searchResults, err := SearchNotes("Batch Sync")
	if err != nil {
		t.Fatalf("SearchNotes failed: %v", err)
	}
	if len(searchResults) < 2 {
		t.Errorf("expected at least 2 search results, got %d", len(searchResults))
	}

	// 2. Verify Tags for Note 1
	tags, err := GetTagsForNote(noteID1)
	if err != nil {
		t.Fatalf("GetTagsForNote failed: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags for note 1, got %d", len(tags))
	}

	// 3. Verify Links
	links, err := GetLinksForNote(noteID1)
	if err != nil {
		t.Fatalf("GetLinksForNote failed: %v", err)
	}
	if len(links) != 1 {
		t.Errorf("expected 1 link for note 1, got %d", len(links))
	}

	// 4. Verify Daily Logs
	log, err := GetDailyLog(logID)
	if err != nil {
		t.Fatalf("GetDailyLog failed: %v", err)
	}
	if log.Content != "Batch synced daily log" {
		t.Errorf("unexpected daily log content: %s", log.Content)
	}

	// 5. Verify Audit Logs
	audit, err := GetAuditEntry(auditID)
	if err != nil {
		t.Fatalf("GetAuditEntry failed: %v", err)
	}
	if audit.EntityID != noteID1 {
		t.Errorf("expected audit entityID %s, got %s", noteID1, audit.EntityID)
	}
}

func TestGetNoteTagsForNotes(t *testing.T) {
	setupTestDB(t)

	n1 := &models.Note{ID: uuid.New().String(), Note: "N1", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	n2 := &models.Note{ID: uuid.New().String(), Note: "N2", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	_ = CreateNote(n1)
	_ = CreateNote(n2)

	_ = AddTag(n1.ID, "tag-alpha")
	_ = AddTag(n1.ID, "tag-beta")
	_ = AddTag(n2.ID, "tag-gamma")

	// Empty query returns nil, nil
	res, err := GetNoteTagsForNotes(nil)
	if err != nil || res != nil {
		t.Fatalf("expected nil for empty slice, got: %v, %v", res, err)
	}

	// Fetch note tags for n1
	res, err = GetNoteTagsForNotes([]string{n1.ID})
	if err != nil {
		t.Fatalf("GetNoteTagsForNotes failed: %v", err)
	}
	if len(res) != 2 {
		t.Errorf("expected 2 note_tags for n1, got %d", len(res))
	}

	// Fetch note tags for both
	res, err = GetNoteTagsForNotes([]string{n1.ID, n2.ID})
	if err != nil {
		t.Fatalf("GetNoteTagsForNotes failed: %v", err)
	}
	if len(res) != 3 {
		t.Errorf("expected 3 note_tags for n1 and n2, got %d", len(res))
	}
}
