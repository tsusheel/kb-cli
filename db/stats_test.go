package db

import (
	"testing"

	"github.com/tsusheel/kb-cli/models"
)

func TestGetKnowledgeBaseStatsAndStreaks(t *testing.T) {
	setupTestDB(t)

	// Create diverse set of notes
	notes := []*models.Note{
		{
			ID:        "11111111111111111111111111111111",
			Note:      "Project Alpha",
			NoteFlesh: "Main project milestone",
			Type:      models.Project,
			Status:    models.InProgress,
			Area:      models.Work,
		},
		{
			ID:        "22222222222222222222222222222222",
			Note:      "Architecture Decision",
			NoteFlesh: "ADR on databases",
			Type:      models.Decision,
			Status:    models.Active,
			Area:      models.Work,
		},
		{
			ID:        "33333333333333333333333333333333",
			Note:      "Personal Todo",
			NoteFlesh: "Buy groceries",
			Type:      models.Todo,
			Status:    models.Completed,
			Area:      models.Personal,
		},
		{
			ID:        "44444444444444444444444444444444",
			Note:      "Orphan Note",
			NoteFlesh: "Unlinked note",
			Type:      models.DefaultNote,
			Status:    models.Active,
		},
	}

	for _, n := range notes {
		if err := CreateNote(n); err != nil {
			t.Fatalf("CreateNote failed: %v", err)
		}
	}

	// Add tags
	_ = AddTag(notes[0].ID, "golang")
	_ = AddTag(notes[1].ID, "golang")
	_ = AddTag(notes[2].ID, "errand")

	// Add link
	_ = AddLink(notes[0].ID, notes[1].ID, models.DependsOn)

	// Add daily log
	_, _ = CreateDailyLog("Daily stream log entry")

	// 1. Test Stats
	stats, err := GetKnowledgeBaseStats()
	if err != nil {
		t.Fatalf("GetKnowledgeBaseStats failed: %v", err)
	}

	if stats.TotalActiveNotes != 4 {
		t.Errorf("TotalActiveNotes = %d, expected 4", stats.TotalActiveNotes)
	}
	if stats.NotesByType["project"] != 1 {
		t.Errorf("NotesByType['project'] = %d, expected 1", stats.NotesByType["project"])
	}
	if stats.NotesByStatus["in-progress"] != 1 {
		t.Errorf("NotesByStatus['in-progress'] = %d, expected 1", stats.NotesByStatus["in-progress"])
	}
	if stats.TotalDailyLogs != 1 {
		t.Errorf("TotalDailyLogs = %d, expected 1", stats.TotalDailyLogs)
	}
	if stats.TodayDailyLogs != 1 {
		t.Errorf("TodayDailyLogs = %d, expected 1", stats.TodayDailyLogs)
	}
	if stats.TotalTags != 2 {
		t.Errorf("TotalTags = %d, expected 2", stats.TotalTags)
	}
	if len(stats.TopTags) == 0 || stats.TopTags[0].Name != "golang" || stats.TopTags[0].Count != 2 {
		t.Errorf("TopTags mismatch: %v", stats.TopTags)
	}
	if len(stats.TopHubNotes) == 0 {
		t.Errorf("expected hub notes for linked notes, got none")
	}

	// 2. Test Orphan notes (note 4 has no links and no tags)
	orphans, err := GetOrphanNotes()
	if err != nil {
		t.Fatalf("GetOrphanNotes failed: %v", err)
	}
	if len(orphans) != 1 || orphans[0].ID != notes[3].ID {
		t.Errorf("expected 1 orphan note (%s), got %v", notes[3].ID, orphans)
	}

	// 3. Test Incoming and Outgoing links
	incoming, err := GetIncomingLinks(notes[1].ID)
	if err != nil {
		t.Fatalf("GetIncomingLinks failed: %v", err)
	}
	if len(incoming) != 1 || incoming[0].FromNote != notes[0].ID {
		t.Errorf("expected incoming link from note 0, got %v", incoming)
	}

	outgoing, err := GetOutgoingLinks(notes[0].ID)
	if err != nil {
		t.Fatalf("GetOutgoingLinks failed: %v", err)
	}
	if len(outgoing) != 1 || outgoing[0].ToNote != notes[1].ID {
		t.Errorf("expected outgoing link to note 1, got %v", outgoing)
	}
}
