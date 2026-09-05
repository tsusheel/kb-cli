package db

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/tsusheel/kb-cli/models"
)

func setupTestDB(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	InitDB(dbPath)
	t.Cleanup(func() {
		CloseDB()
	})
	if err := RunMigrations(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}
}

func TestCreateAndGetNote(t *testing.T) {
	setupTestDB(t)

	now := time.Now().Truncate(time.Second)
	dueDate := now.AddDate(0, 0, 3)

	n := &models.Note{
		ID:             "1234567890abcdef1234567890abcdef",
		Note:           "Test Task Note",
		NoteFlesh:      "This is the detailed body of the note.",
		Type:           models.Todo,
		Status:         models.Active,
		Area:           models.Work,
		Importance:     4,
		Clarity:        5,
		Source:         "CLI Unit Test",
		TargetDateTime: dueDate,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := CreateNote(n); err != nil {
		t.Fatalf("CreateNote failed: %v", err)
	}

	// Test ResolveID and GetNote
	got, err := GetNote("1234567")
	if err != nil {
		t.Fatalf("GetNote failed: %v", err)
	}

	if got.Note != n.Note {
		t.Errorf("Note = %q, expected %q", got.Note, n.Note)
	}
	if got.NoteFlesh != n.NoteFlesh {
		t.Errorf("NoteFlesh = %q, expected %q", got.NoteFlesh, n.NoteFlesh)
	}
	if got.Type != models.Todo {
		t.Errorf("Type = %q, expected %q", got.Type, models.Todo)
	}
	if got.Status != models.Active {
		t.Errorf("Status = %q, expected %q", got.Status, models.Active)
	}
	if got.Importance != 4 || got.Clarity != 5 {
		t.Errorf("Importance/Clarity mismatch: %d, %d", got.Importance, got.Clarity)
	}
	if got.TargetDateTime.IsZero() || !got.TargetDateTime.Equal(dueDate) {
		t.Errorf("TargetDateTime = %v, expected %v", got.TargetDateTime, dueDate)
	}
}

func TestListAndSearchNotes(t *testing.T) {
	setupTestDB(t)

	n1 := &models.Note{
		ID:        "aaaa1111222233334444555566667777",
		Note:      "Golang Knowledge Base",
		NoteFlesh: "A CLI tool built with Go and SQLite",
		Type:      models.Project,
		Status:    models.Active,
	}
	n2 := &models.Note{
		ID:        "bbbb1111222233334444555566667777",
		Note:      "Buy Groceries",
		NoteFlesh: "Milk, eggs, coffee beans",
		Type:      models.Todo,
		Status:    models.Active,
	}

	if err := CreateNote(n1); err != nil {
		t.Fatalf("CreateNote n1 failed: %v", err)
	}
	if err := CreateNote(n2); err != nil {
		t.Fatalf("CreateNote n2 failed: %v", err)
	}

	// Test listing all
	allNotes, err := ListNotes("")
	if err != nil {
		t.Fatalf("ListNotes('') failed: %v", err)
	}
	if len(allNotes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(allNotes))
	}

	// Test listing by type
	todos, err := ListNotes("todo")
	if err != nil {
		t.Fatalf("ListNotes('todo') failed: %v", err)
	}
	if len(todos) != 1 || todos[0].Note != "Buy Groceries" {
		t.Errorf("expected 1 todo note, got %v", todos)
	}

	// Test FTS search
	searchResults, err := SearchNotes("SQLite")
	if err != nil {
		t.Fatalf("SearchNotes('SQLite') failed: %v", err)
	}
	if len(searchResults) != 1 || searchResults[0].ID != n1.ID {
		t.Errorf("expected search result for 'SQLite', got %v", searchResults)
	}
}

func TestTagsAndLinks(t *testing.T) {
	setupTestDB(t)

	n1 := &models.Note{
		ID:        "11111111111111111111111111111111",
		Note:      "First Note",
		NoteFlesh: "First note flesh",
		Type:      models.DefaultNote,
		Status:    models.Active,
	}
	n2 := &models.Note{
		ID:        "22222222222222222222222222222222",
		Note:      "Second Note",
		NoteFlesh: "Second note flesh",
		Type:      models.DefaultNote,
		Status:    models.Active,
	}

	if err := CreateNote(n1); err != nil {
		t.Fatal(err)
	}
	if err := CreateNote(n2); err != nil {
		t.Fatal(err)
	}

	if err := AddTag(n1.ID, "golang/cli"); err != nil {
		t.Fatalf("AddTag failed: %v", err)
	}

	tags, err := GetTagsForNote(n1.ID)
	if err != nil {
		t.Fatalf("GetTagsForNote failed: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "golang/cli" {
		t.Errorf("unexpected tags: %v", tags)
	}

	if err := AddLink(n1.ID, n2.ID, models.RelatedTo); err != nil {
		t.Fatalf("AddLink failed: %v", err)
	}

	links, err := GetLinksForNote(n1.ID)
	if err != nil {
		t.Fatalf("GetLinksForNote failed: %v", err)
	}
	if len(links) != 1 || links[0].Type != models.RelatedTo {
		t.Errorf("unexpected links: %v", links)
	}
}

func TestUpdateAndSoftDeleteNote(t *testing.T) {
	setupTestDB(t)

	n := &models.Note{
		ID:        "cccc1111222233334444555566667777",
		Note:      "Original Title",
		NoteFlesh: "Original Flesh",
		Type:      models.DefaultNote,
		Status:    models.Active,
		Area:      models.Personal,
	}

	if err := CreateNote(n); err != nil {
		t.Fatal(err)
	}

	// Update note
	n.Note = "Updated Title"
	n.NoteFlesh = "Updated Flesh Content"
	n.Status = models.InProgress
	if err := UpdateNote(n); err != nil {
		t.Fatalf("UpdateNote failed: %v", err)
	}

	got, err := GetNote(n.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Note != "Updated Title" || got.Status != models.InProgress {
		t.Errorf("updated note mismatch: %v", got)
	}

	// Soft delete note
	if err := SoftDeleteNote(n.ID, "deleted by AI for testing"); err != nil {
		t.Fatalf("SoftDeleteNote failed: %v", err)
	}

	// Verify excluded from normal list
	activeNotes, err := ListNotes("")
	if err != nil {
		t.Fatal(err)
	}
	if len(activeNotes) != 0 {
		t.Errorf("expected 0 active notes, got %d", len(activeNotes))
	}

	// Verify included when includeDeleted is true
	allNotes, err := ListNotesExtended("", "", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(allNotes) != 1 || allNotes[0].DeletedNote != "deleted by AI for testing" {
		t.Errorf("expected 1 deleted note with reason, got %v", allNotes)
	}

	// Verify FTS excludes deleted notes
	searchResults, err := SearchNotes("Updated")
	if err != nil {
		t.Fatal(err)
	}
	if len(searchResults) != 0 {
		t.Errorf("expected 0 search results for deleted note, got %d", len(searchResults))
	}
}
