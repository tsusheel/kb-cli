package cmd

import (
	"path/filepath"
	"testing"

	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
)

func setupCmdTestDB(t *testing.T) {
	_ = db.CloseDB()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "cmd_test.db")
	db.InitDB(dbPath)
	t.Cleanup(func() {
		_ = db.CloseDB()
	})
	if err := db.InitSchema(); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}
}

func TestExportAndImportRoundtrip(t *testing.T) {
	setupCmdTestDB(t)

	// 1. Create sample notes
	n1 := &models.Note{
		Note:      "Exported Project Idea",
		NoteFlesh: "Detailed markdown description of the exported idea.",
		Type:      models.Idea,
		Status:    models.Active,
		Area:      models.Work,
	}
	if err := db.CreateNote(n1); err != nil {
		t.Fatalf("CreateNote n1 failed: %v", err)
	}
	_ = db.AddTag(n1.ID, "dev/golang")
	_ = db.AddTag(n1.ID, "architecture")

	_, _ = db.CreateDailyLog("Sample daily log for export")

	// 2. Export to temp folder
	exportTarget := filepath.Join(t.TempDir(), "export-out")
	exportDir = exportTarget
	exportFormat = "markdown"
	exportIncludeDeleted = false

	if err := exportCmd.RunE(exportCmd, nil); err != nil {
		t.Fatalf("exportCmd failed: %v", err)
	}

	// 3. Import from exported notes folder into a fresh DB
	setupCmdTestDB(t)

	importNotesDir := filepath.Join(exportTarget, "notes")
	importDir = importNotesDir

	if err := importCmd.RunE(importCmd, nil); err != nil {
		t.Fatalf("importCmd failed: %v", err)
	}

	// 4. Verify imported note in fresh DB
	importedNotes, err := db.ListNotes("")
	if err != nil {
		t.Fatalf("ListNotes failed: %v", err)
	}
	if len(importedNotes) != 1 {
		t.Fatalf("expected 1 imported note, got %d", len(importedNotes))
	}
	if importedNotes[0].Note != "Exported Project Idea" {
		t.Errorf("imported title mismatch: %q", importedNotes[0].Note)
	}
	if importedNotes[0].Type != models.Idea {
		t.Errorf("imported type mismatch: %q", importedNotes[0].Type)
	}

	tags, _ := db.GetTagsForNote(importedNotes[0].ID)
	if len(tags) != 2 {
		t.Errorf("expected 2 tags for imported note, got %d", len(tags))
	}
}

func TestParseMarkdownNote(t *testing.T) {
	docWithFrontmatter := `---
title: "Custom Document"
type: "decision"
status: "refined"
area: "finance"
tags: ["investing", "crypto"]
---

# Custom Document

Here is the markdown flesh body.
`
	fm, body := parseMarkdownNote(docWithFrontmatter)
	if fm.Title != "Custom Document" {
		t.Errorf("Title = %q, expected 'Custom Document'", fm.Title)
	}
	if fm.Type != "decision" {
		t.Errorf("Type = %q, expected 'decision'", fm.Type)
	}
	if len(fm.Tags) != 2 || fm.Tags[0] != "investing" {
		t.Errorf("Tags mismatch: %v", fm.Tags)
	}
	if body != "Here is the markdown flesh body." {
		t.Errorf("Body = %q", body)
	}

	docWithoutFrontmatter := `# Header Only Note

Simple plain text content without yaml.
`
	fm2, body2 := parseMarkdownNote(docWithoutFrontmatter)
	if fm2.Title != "Header Only Note" {
		t.Errorf("Title = %q, expected 'Header Only Note'", fm2.Title)
	}
	if body2 != "Simple plain text content without yaml." {
		t.Errorf("Body = %q", body2)
	}
}
