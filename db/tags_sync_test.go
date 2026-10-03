package db

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tsusheel/kb-cli/models"
)

func setupTagsTestDB(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_tags.db")
	if err := InitDB(dbPath); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	t.Cleanup(func() {
		CloseDB()
	})
	if err := InitSchema(); err != nil {
		t.Fatalf("InitSchema failed: %v", err)
	}
}

func TestAddRemoveTag_UpdatesNoteTimestamp(t *testing.T) {
	setupTagsTestDB(t)

	noteID := uuid.New().String()
	initialTime := time.Now().Add(-1 * time.Hour)
	note := &models.Note{
		ID:        noteID,
		Note:      "Test Note for Tagging",
		Type:      models.DefaultNote,
		Status:    models.Active,
		CreatedAt: initialTime,
		UpdatedAt: initialTime,
	}

	if err := CreateNote(note); err != nil {
		t.Fatalf("CreateNote failed: %v", err)
	}

	// Add Tag
	if err := AddTag(noteID, "golang"); err != nil {
		t.Fatalf("AddTag failed: %v", err)
	}

	updatedNote, err := GetNote(noteID)
	if err != nil {
		t.Fatalf("GetNote failed: %v", err)
	}
	if !updatedNote.UpdatedAt.After(initialTime) {
		t.Errorf("expected UpdatedAt after initialTime, got %v", updatedNote.UpdatedAt)
	}

	// Remove Tag
	beforeRemove := updatedNote.UpdatedAt
	time.Sleep(10 * time.Millisecond)
	if err := RemoveTag(noteID, "golang"); err != nil {
		t.Fatalf("RemoveTag failed: %v", err)
	}

	afterRemoveNote, err := GetNote(noteID)
	if err != nil {
		t.Fatalf("GetNote failed: %v", err)
	}
	if !afterRemoveNote.UpdatedAt.After(beforeRemove) {
		t.Errorf("expected UpdatedAt after beforeRemove, got %v", afterRemoveNote.UpdatedAt)
	}
}

func TestUpsertRemoteTag_HarmonizesTagIDs(t *testing.T) {
	setupTagsTestDB(t)

	noteID := uuid.New().String()
	note := &models.Note{
		ID:        noteID,
		Note:      "Harmonization test note",
		Type:      models.DefaultNote,
		Status:    models.Active,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := CreateNote(note); err != nil {
		t.Fatalf("CreateNote failed: %v", err)
	}

	// Locally add tag "architecture" -> generates local UUID
	if err := AddTag(noteID, "architecture"); err != nil {
		t.Fatalf("AddTag failed: %v", err)
	}

	localTags, err := GetTagsForNote(noteID)
	if err != nil || len(localTags) != 1 {
		t.Fatalf("expected 1 tag, got: %v", localTags)
	}
	localTagID := localTags[0].ID

	// Remote tag arrives with different ID for "architecture"
	remoteTagID := "remote-canonical-uuid-123"
	remoteTag := &models.Tag{
		ID:        remoteTagID,
		Name:      "architecture",
		CreatedAt: time.Now(),
	}

	if err := UpsertRemoteTag(remoteTag); err != nil {
		t.Fatalf("UpsertRemoteTag failed: %v", err)
	}

	// Note's tags should still be found via JOIN with the harmonized remote ID
	harmonizedTags, err := GetTagsForNote(noteID)
	if err != nil {
		t.Fatalf("GetTagsForNote failed: %v", err)
	}
	if len(harmonizedTags) != 1 {
		t.Fatalf("expected 1 tag after harmonization, got %d", len(harmonizedTags))
	}
	if harmonizedTags[0].ID != remoteTagID {
		t.Errorf("expected tag ID %s, got %s (was %s)", remoteTagID, harmonizedTags[0].ID, localTagID)
	}
}

func TestUpsertRemoteLink_ConflictResolution(t *testing.T) {
	setupTagsTestDB(t)

	n1 := &models.Note{ID: uuid.New().String(), Note: "N1", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	n2 := &models.Note{ID: uuid.New().String(), Note: "N2", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	_ = CreateNote(n1)
	_ = CreateNote(n2)

	linkID := uuid.New().String()
	link := &models.Link{
		ID:        linkID,
		FromNote:  n1.ID,
		ToNote:    n2.ID,
		Type:      models.RelatedTo,
		CreatedAt: time.Now(),
	}

	if err := UpsertRemoteLink(link); err != nil {
		t.Fatalf("UpsertRemoteLink failed: %v", err)
	}

	// Soft delete locally
	if err := SoftDeleteLink(linkID, "user deleted"); err != nil {
		t.Fatalf("SoftDeleteLink failed: %v", err)
	}

	// Upserting non-deleted remote link shouldn't resurrect locally deleted link
	if err := UpsertRemoteLink(link); err != nil {
		t.Fatalf("UpsertRemoteLink failed: %v", err)
	}

	links, err := GetLinksForNote(n1.ID)
	if err != nil {
		t.Fatalf("GetLinksForNote failed: %v", err)
	}
	if len(links) != 0 {
		t.Errorf("expected 0 active links after soft delete, got %d", len(links))
	}
}
