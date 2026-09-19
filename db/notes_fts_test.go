package db

import (
	"testing"

	"github.com/tsusheel/kb-cli/models"
)

func TestSanitizeFTS5Query(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "simple single word",
			input:    "golang",
			expected: `"golang"*`,
		},
		{
			name:     "multiple words",
			input:    "distributed raft consensus",
			expected: `"distributed"* "raft"* "consensus"*`,
		},
		{
			name:     "special characters c++",
			input:    "c++",
			expected: `"c"*`,
		},
		{
			name:     "path or slash foo/bar",
			input:    "foo/bar",
			expected: `"foo/bar"*`,
		},
		{
			name:     "quotes and punctuation (query)",
			input:    "(sqlite) & database",
			expected: `"sqlite"* "&"* "database"*`,
		},
		{
			name:     "explicit phrase match",
			input:    `"hello world"`,
			expected: `"hello world"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeFTS5Query(tc.input)
			if got != tc.expected {
				t.Errorf("SanitizeFTS5Query(%q) = %q, expected %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestSearchNotesSpecialCharacters(t *testing.T) {
	setupTestDB(t)

	notes := []*models.Note{
		{
			Note:      "Learning C++ and Systems Programming",
			NoteFlesh: "Exploring memory management, pointers, and RAII in Modern C++",
			Type:      models.DefaultNote,
			Status:    models.Active,
		},
		{
			Note:      "Setting up CI/CD pipeline with GitHub Actions",
			NoteFlesh: "Automating go test ./... and lint checks on pull requests",
			Type:      models.Project,
			Status:    models.Active,
		},
		{
			Note:      "Architecture Note (Decision): Use SQLite + Postgres",
			NoteFlesh: "Local-first SQLite for latency, PostgreSQL for remote backup",
			Type:      models.Decision,
			Status:    models.Active,
		},
	}

	for _, n := range notes {
		if err := CreateNote(n); err != nil {
			t.Fatalf("CreateNote failed: %v", err)
		}
	}

	specialQueries := []string{
		"C++",
		"CI/CD",
		"(Decision)",
		"go test",
		"SQLite + Postgres",
		"***###",
		`"Modern C++"`,
	}

	for _, query := range specialQueries {
		t.Run("query_"+query, func(t *testing.T) {
			results, err := SearchNotes(query)
			if err != nil {
				t.Fatalf("SearchNotes(%q) produced unexpected error: %v", query, err)
			}
			t.Logf("Search %q returned %d results", query, len(results))
		})
	}
}

func TestRevertNoteSingleAuditLog(t *testing.T) {
	setupTestDB(t)

	n := &models.Note{
		Note:      "Initial Note Title",
		NoteFlesh: "Initial Body Content",
		Type:      models.DefaultNote,
		Status:    models.Active,
	}
	if err := CreateNote(n); err != nil {
		t.Fatalf("CreateNote failed: %v", err)
	}

	// 1. Fetch initial audit snapshot
	history, err := GetAuditHistory("note", n.ID, 10)
	if err != nil || len(history) != 1 {
		t.Fatalf("expected 1 audit entry for creation, got %d (err: %v)", len(history), err)
	}
	initialSnapshot := history[0].SnapshotJSON

	// 2. Update note to change content
	n.Note = "Modified Note Title"
	n.NoteFlesh = "Modified Body Content"
	if err := UpdateNote(n); err != nil {
		t.Fatalf("UpdateNote failed: %v", err)
	}

	historyAfterUpdate, err := GetAuditHistory("note", n.ID, 10)
	if err != nil || len(historyAfterUpdate) != 2 {
		t.Fatalf("expected 2 audit entries after update, got %d", len(historyAfterUpdate))
	}

	// 3. Revert note to initial snapshot
	reverted, err := RevertNoteToSnapshot(n.ID, initialSnapshot)
	if err != nil {
		t.Fatalf("RevertNoteToSnapshot failed: %v", err)
	}
	if reverted.Note != "Initial Note Title" {
		t.Errorf("expected reverted title 'Initial Note Title', got %q", reverted.Note)
	}

	// 4. Verify exactly 3 audit entries total (create, update, restored - NOT 4)
	historyAfterRevert, err := GetAuditHistory("note", n.ID, 10)
	if err != nil || len(historyAfterRevert) != 3 {
		t.Fatalf("expected exactly 3 audit entries after revert (create, update, restore), got %d: %v", len(historyAfterRevert), historyAfterRevert)
	}

	if historyAfterRevert[0].Action != models.ActionRestored {
		t.Errorf("expected latest action to be ActionRestored, got %s", historyAfterRevert[0].Action)
	}
}
