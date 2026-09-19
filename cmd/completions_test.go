package cmd

import (
	"strings"
	"testing"
)

func TestCLIItemFormatFuzzy(t *testing.T) {
	item := CLIItem{
		ID:        "1234567890abcdef",
		Type:      "note",
		Status:    "raw",
		Area:      "work",
		Display:   "Refactor database layer",
		Flesh:     "Add connection pooling and retry policies for PostgreSQL",
		Tags:      []string{"db", "postgres"},
		Timestamp: "2026-09-19 11:00",
		IsLog:     false,
	}

	formatted := item.FormatFuzzy()

	// Verify all searchable components are present in the formatted single line string
	checks := []string{
		"1234567",
		"2026-09-19 11:00",
		"note:raw",
		"work",
		"#db #postgres",
		"Refactor database layer",
		"connection pooling and retry policies",
	}

	for _, c := range checks {
		if !strings.Contains(formatted, c) {
			t.Errorf("expected FormatFuzzy() to contain %q, got:\n%s", c, formatted)
		}
	}
}

func TestCLIItemRenderPreview(t *testing.T) {
	noteItem := CLIItem{
		ID:        "1234567890abcdef",
		Type:      "todo",
		Status:    "in-progress",
		Area:      "engineering",
		Display:   "Implement Kafka consumer",
		Flesh:     "Partition consumer with backoff logic and dead-letter queues.",
		Tags:      []string{"kafka", "streaming"},
		Timestamp: "2026-09-19 11:00",
		IsLog:     false,
	}

	preview := noteItem.RenderPreview()
	if !strings.Contains(preview, "=== NOTE [1234567] ===") {
		t.Errorf("expected header in preview, got:\n%s", preview)
	}
	if !strings.Contains(preview, "Title    : Implement Kafka consumer") {
		t.Errorf("expected title in preview, got:\n%s", preview)
	}
	if !strings.Contains(preview, "Partition consumer with backoff logic") {
		t.Errorf("expected flesh in preview, got:\n%s", preview)
	}
	if !strings.Contains(preview, "#kafka #streaming") {
		t.Errorf("expected tags in preview, got:\n%s", preview)
	}

	logItem := CLIItem{
		ID:        "abcdef1234567890",
		Type:      "log:promoted",
		Display:   "Morning standup notes",
		Timestamp: "2026-09-19 09:30",
		IsLog:     true,
	}

	logPreview := logItem.RenderPreview()
	if !strings.Contains(logPreview, "=== DAILY LOG [abcdef1] ===") {
		t.Errorf("expected log header in preview, got:\n%s", logPreview)
	}
	if !strings.Contains(logPreview, "Status   : Promoted to Note") {
		t.Errorf("expected promoted status in log preview, got:\n%s", logPreview)
	}
}
