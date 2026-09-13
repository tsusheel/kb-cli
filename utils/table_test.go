package utils

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/tsusheel/kb-cli/models"
)

func TestRenderNotesTable(t *testing.T) {
	notes := []models.Note{
		{
			ID:        "12345678-abcd-ef01-2345-6789abcdef01",
			Note:      "Short note",
			Type:      models.DefaultNote,
			Status:    models.Active,
			UpdatedAt: time.Date(2026, 9, 13, 10, 30, 0, 0, time.UTC),
		},
		{
			ID:        "87654321-dcba-fe10-5432-10fedcba9876",
			Note:      "This is a longer note designed to test automatic line wrapping across multiple rows in tablewriter output.",
			Type:      models.Todo,
			Status:    models.Raw,
			UpdatedAt: time.Date(2026, 9, 13, 11, 45, 0, 0, time.UTC),
		},
	}

	var buf bytes.Buffer
	RenderNotesTable(notes, &buf)

	output := buf.String()
	if !strings.Contains(output, "ID") || !strings.Contains(output, "NOTE") || !strings.Contains(output, "STATUS") {
		t.Errorf("expected table header in output, got:\n%s", output)
	}
	if !strings.Contains(output, "1234567") {
		t.Errorf("expected short ID '1234567' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Short note") {
		t.Errorf("expected 'Short note' in output, got:\n%s", output)
	}
}

func TestRenderNotesTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	RenderNotesTable([]models.Note{}, &buf)

	output := strings.TrimSpace(buf.String())
	if output != "No notes found." {
		t.Errorf("expected 'No notes found.', got %q", output)
	}
}

func TestRenderDailyLogsTable(t *testing.T) {
	logs := []models.DailyLog{
		{
			ID:        "d1664d61234567890abcdef123456789",
			Content:   "First log entry of the day",
			NoteID:    "d8d1e071234567890abcdef123456789",
			CreatedAt: time.Date(2026, 9, 13, 5, 44, 0, 0, time.UTC),
		},
		{
			ID:        "a9876541234567890abcdef123456789",
			Content:   "Unpromoted quick thought",
			CreatedAt: time.Date(2026, 9, 13, 6, 15, 0, 0, time.UTC),
		},
	}

	var buf bytes.Buffer
	RenderDailyLogsTable(logs, &buf)

	output := buf.String()
	if !strings.Contains(output, "ID") || !strings.Contains(output, "TIME") || !strings.Contains(output, "CONTENT") || !strings.Contains(output, "STATUS") {
		t.Errorf("expected daily log headers, got:\n%s", output)
	}
	if !strings.Contains(output, "d1664d6") {
		t.Errorf("expected short ID 'd1664d6', got:\n%s", output)
	}
	if !strings.Contains(output, "promoted (d8d1e07)") {
		t.Errorf("expected 'promoted (d8d1e07)', got:\n%s", output)
	}
}

func TestRenderNoteDetail(t *testing.T) {
	note := &models.Note{
		ID:        "12345678-abcd-ef01-2345-6789abcdef01",
		Note:      "Design System Overview",
		NoteFlesh: "### Architecture\nDetailed documentation body.",
		Type:      models.Concept,
		Status:    models.Refined,
		Area:      models.Area("tech"),
		Importance: 4,
		Clarity:    5,
		Source:     "Team RFC",
		CreatedAt: time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 9, 13, 11, 30, 0, 0, time.UTC),
	}
	tags := []models.Tag{{Name: "architecture"}, {Name: "design"}}
	links := []models.Link{{FromNote: note.ID, ToNote: "87654321-dcba-fe10-5432-10fedcba9876", Type: models.RelatedTo}}

	var buf bytes.Buffer
	RenderNoteDetail(note, tags, links, &buf)

	output := buf.String()
	if !strings.Contains(output, "1234567") {
		t.Errorf("expected short ID '1234567', got:\n%s", output)
	}
	if !strings.Contains(output, "Design System Overview") {
		t.Errorf("expected title 'Design System Overview', got:\n%s", output)
	}
	if !strings.Contains(output, "#architecture") {
		t.Errorf("expected tag '#architecture', got:\n%s", output)
	}
	if !strings.Contains(output, "BODY CONTENT") {
		t.Errorf("expected 'BODY CONTENT' header, got:\n%s", output)
	}
}


