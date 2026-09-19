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

func TestRenderConfigTable(t *testing.T) {
	items := []ConfigItem{
		{
			Key:    "editor",
			Value:  "code --wait",
			Source: "config.yaml",
		},
		{
			Key:    "postgres_password",
			Value:  "[ENCRYPTED / SECURE]",
			Source: "OS Keyring",
		},
	}

	var buf bytes.Buffer
	RenderConfigTable(items, &buf)

	output := buf.String()
	if !strings.Contains(output, "KEY") || !strings.Contains(output, "VALUE") || !strings.Contains(output, "SOURCE") {
		t.Errorf("expected config table headers, got:\n%s", output)
	}
	if !strings.Contains(output, "editor") || !strings.Contains(output, "code --wait") {
		t.Errorf("expected editor config row, got:\n%s", output)
	}
	if !strings.Contains(output, "postgres_password") || !strings.Contains(output, "[ENCRYPTED / SECURE]") {
		t.Errorf("expected secret config row, got:\n%s", output)
	}
}

func TestRenderConfigTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	RenderConfigTable([]ConfigItem{}, &buf)

	output := strings.TrimSpace(buf.String())
	if output != "No configuration found." {
		t.Errorf("expected 'No configuration found.', got %q", output)
	}
}

func TestRenderSyncStatusTable(t *testing.T) {
	info := SyncStatusInfo{
		Enabled:       false,
		Provider:      "PostgreSQL",
		DatabaseURL:   "postgresql://postgres@localhost:5432/kb",
		HasPassword:   true,
		LastSyncedAt:  time.Date(2026, 9, 13, 14, 0, 0, 0, time.UTC),
		UnsyncedNotes: 2,
		UnsyncedLogs:  1,
	}

	var buf bytes.Buffer
	RenderSyncStatusTable(info, &buf)

	output := buf.String()
	if !strings.Contains(output, "PROPERTY") || !strings.Contains(output, "VALUE") {
		t.Errorf("expected sync status table headers, got:\n%s", output)
	}
	if !strings.Contains(output, "Sync Status") || !strings.Contains(output, "DISABLED (remote.enabled = false)") {
		t.Errorf("expected disabled sync status in table, got:\n%s", output)
	}
	if !strings.Contains(output, "Database URL") || !strings.Contains(output, "postgresql://postgres@localhost:5432/kb") {
		t.Errorf("expected database url in table, got:\n%s", output)
	}
	if !strings.Contains(output, "Pending Changes") || !strings.Contains(output, "2 un-synced notes, 1 un-synced logs") {
		t.Errorf("expected pending changes count in table, got:\n%s", output)
	}
}

func TestRenderAuditTable(t *testing.T) {
	entries := []models.AuditEntry{
		{
			ID:             "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4",
			EntityType:     "note",
			EntityID:       "1234567890abcdef1234567890abcdef",
			Action:         models.ActionUpdated,
			ChangesSummary: "status: raw -> active, title updated",
			CreatedAt:      time.Date(2026, 9, 13, 15, 30, 0, 0, time.UTC),
		},
		{
			ID:             "b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5",
			EntityType:     "note",
			EntityID:       "1234567890abcdef1234567890abcdef",
			Action:         models.ActionCreated,
			ChangesSummary: "created note",
			CreatedAt:      time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC),
		},
	}

	var buf bytes.Buffer
	RenderAuditTable(entries, "=== Audit History ===", &buf)

	output := buf.String()
	if !strings.Contains(output, "REV") || !strings.Contains(output, "TIME") || !strings.Contains(output, "ACTION") || !strings.Contains(output, "CHANGES") {
		t.Errorf("expected audit table headers, got:\n%s", output)
	}
	if !strings.Contains(output, "a1b2c3d") || !strings.Contains(output, "status: raw -> active") {
		t.Errorf("expected audit row content, got:\n%s", output)
	}
}

func TestRenderBacklinksAndOrphansTable(t *testing.T) {
	links := []models.Link{
		{
			ID:        "link1",
			FromNote:  "11111111111111111111111111111111",
			ToNote:    "22222222222222222222222222222222",
			Type:      models.DependsOn,
			CreatedAt: time.Now(),
		},
	}
	noteMap := map[string]*models.Note{
		"11111111111111111111111111111111": {ID: "11111111111111111111111111111111", Note: "Source Feature"},
	}

	var buf bytes.Buffer
	RenderBacklinksTable(links, noteMap, &buf)
	if !strings.Contains(buf.String(), "Source Feature") || !strings.Contains(buf.String(), "depends_on") {
		t.Errorf("expected backlinks table output, got: %s", buf.String())
	}

	var orphanBuf bytes.Buffer
	orphans := []models.Note{
		{ID: "33333333333333333333333333333333", Note: "Lonely Note", Type: models.DefaultNote, UpdatedAt: time.Now()},
	}
	RenderOrphansTable(orphans, &orphanBuf)
	if !strings.Contains(orphanBuf.String(), "Lonely Note") {
		t.Errorf("expected orphan table output, got: %s", orphanBuf.String())
	}
}

func TestRenderStatsAndGraph(t *testing.T) {
	stats := &models.KBStats{
		TotalActiveNotes:  15,
		TotalDeletedNotes: 2,
		NotesByType:       map[string]int{"project": 3, "todo": 5, "note": 7},
		TotalDailyLogs:    20,
		TodayDailyLogs:    3,
		ConsecutiveStreak: 4,
	}

	var statsBuf bytes.Buffer
	RenderStatsDashboard(stats, &statsBuf)
	if !strings.Contains(statsBuf.String(), "Total Active Notes") || !strings.Contains(statsBuf.String(), "15") {
		t.Errorf("expected stats output, got: %s", statsBuf.String())
	}

	rootNote := &models.Note{
		ID:   "root1234567890abcdef1234567890",
		Note: "Root Project",
		Type: models.Project,
	}
	outgoing := []models.Link{
		{FromNote: rootNote.ID, ToNote: "target1234", Type: models.RelatedTo},
	}
	noteMap := map[string]*models.Note{
		"target1234": {ID: "target1234", Note: "Subtask Note"},
	}

	var graphBuf bytes.Buffer
	RenderGraphTree(rootNote, outgoing, nil, noteMap, &graphBuf)
	if !strings.Contains(graphBuf.String(), "Root Project") || !strings.Contains(graphBuf.String(), "Subtask Note") {
		t.Errorf("expected graph tree output, got: %s", graphBuf.String())
	}
}

