package db

import (
	"path/filepath"
	"testing"

	"github.com/tsusheel/kb-cli/models"
)

func TestPluginRegistrationAndCapabilities(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_plugin_reg.db")

	if err := InitDB(dbPath); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer CloseDB()

	if err := InitSchema(); err != nil {
		t.Fatalf("InitSchema failed: %v", err)
	}

	rec := &models.PluginRecord{
		PluginID:      "test_plugin",
		Version:       "1.0.0",
		APIVersion:    "1",
		SchemaVersion: 1,
		Enabled:       false,
		Source:        "local_bin",
		ManifestJSON:  `{"id":"test_plugin","name":"Test Plugin"}`,
	}

	if err := RegisterPluginRecord(rec); err != nil {
		t.Fatalf("RegisterPluginRecord failed: %v", err)
	}

	fetched, err := GetPluginRecord("test_plugin")
	if err != nil {
		t.Fatalf("GetPluginRecord failed: %v", err)
	}
	if fetched.PluginID != "test_plugin" || fetched.Enabled {
		t.Errorf("unexpected record: %+v", fetched)
	}

	if err := SetPluginEnabled("test_plugin", true); err != nil {
		t.Fatalf("SetPluginEnabled failed: %v", err)
	}

	fetched, _ = GetPluginRecord("test_plugin")
	if !fetched.Enabled {
		t.Errorf("expected plugin to be enabled")
	}

	// Capabilities
	hasCap, err := HasPluginCapability("test_plugin", "storage.entries")
	if err != nil {
		t.Fatalf("HasPluginCapability failed: %v", err)
	}
	if hasCap {
		t.Errorf("expected capability to be false initially")
	}

	if err := SetPluginCapability("test_plugin", "storage.entries", true); err != nil {
		t.Fatalf("SetPluginCapability failed: %v", err)
	}

	hasCap, _ = HasPluginCapability("test_plugin", "storage.entries")
	if !hasCap {
		t.Errorf("expected capability to be true after grant")
	}

	caps, err := GetPluginCapabilities("test_plugin")
	if err != nil || len(caps) != 1 || caps[0] != "storage.entries" {
		t.Errorf("unexpected capabilities list: %v, err: %v", caps, err)
	}
}

func TestUniversalEntryLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_plugin_store.db")

	if err := InitDB(dbPath); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer CloseDB()

	if err := InitSchema(); err != nil {
		t.Fatalf("InitSchema failed: %v", err)
	}

	// Register plugin first (foreign key reference)
	_ = RegisterPluginRecord(&models.PluginRecord{
		PluginID:      "gym",
		Version:       "1.0.0",
		APIVersion:    "1",
		SchemaVersion: 1,
		Enabled:       true,
		Source:        "builtin",
		ManifestJSON:  `{"id":"gym"}`,
	})

	dateStr := "2026-10-20"
	entry := &models.PluginEntry{
		PluginID:   "gym",
		EntityType: "workout_set",
		EntryDate:  &dateStr,
		NumVal:     70.0,
		SearchText: "bench press chest workout",
		Data:       `{"exercise":"bench_press","weight":70,"reps":8}`,
	}

	actor := models.DefaultUserActor()

	// 1. Create Entry
	if err := CreateEntry(nil, entry, actor); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if entry.ID == "" {
		t.Fatalf("expected generated ID")
	}
	if entry.Revision != 1 {
		t.Errorf("expected initial revision 1, got %d", entry.Revision)
	}

	// Check short prefix resolution
	shortPrefix := entry.ID[:8]
	resolvedID, err := ResolvePluginEntryID("gym", shortPrefix)
	if err != nil || resolvedID != entry.ID {
		t.Fatalf("failed resolving short ID prefix %s: %v", shortPrefix, err)
	}

	// Verify revision snapshot recorded
	revs, err := GetEntityRevisions("gym", "workout_set", entry.ID, 10)
	if err != nil {
		t.Fatalf("GetEntityRevisions failed: %v", err)
	}
	if len(revs) != 1 || revs[0].Revision != 1 {
		t.Fatalf("expected 1 revision snapshot with rev 1, got %+v", revs)
	}

	// Verify audit log recorded
	auditHistory, err := GetAuditHistory("workout_set", entry.ID, 10)
	if err != nil {
		t.Fatalf("GetAuditHistory failed: %v", err)
	}
	if len(auditHistory) != 1 || auditHistory[0].Action != models.ActionCreated {
		t.Fatalf("expected audit creation event, got %+v", auditHistory)
	}

	// Verify outbox entry
	var outboxCount int
	var op string
	_ = DB.QueryRow(`SELECT count(*), operation FROM sync_outbox WHERE plugin_id = 'gym' AND entity_id = ?`, entry.ID).Scan(&outboxCount, &op)
	if outboxCount != 1 || op != "upsert" {
		t.Errorf("expected 1 outbox upsert entry, got count=%d, op=%s", outboxCount, op)
	}

	// 2. Optimistic Concurrency Update
	entry.NumVal = 75.0
	entry.Data = `{"exercise":"bench_press","weight":75,"reps":8}`

	// Stale revision update should fail
	if err := UpdateEntry(nil, entry, 999, actor); err == nil {
		t.Fatalf("expected revision conflict error for stale expected revision, got nil")
	}

	// Correct revision update
	if err := UpdateEntry(nil, entry, 1, actor); err != nil {
		t.Fatalf("UpdateEntry failed: %v", err)
	}
	if entry.Revision != 2 {
		t.Errorf("expected revision 2, got %d", entry.Revision)
	}

	revs, _ = GetEntityRevisions("gym", "workout_set", entry.ID, 10)
	if len(revs) != 2 {
		t.Errorf("expected 2 revisions, got %d", len(revs))
	}

	// 3. Query with FTS5 search
	results, err := QueryEntries(models.EntryFilter{
		PluginID:   "gym",
		SearchText: "bench",
	})
	if err != nil {
		t.Fatalf("QueryEntries with search failed: %v", err)
	}
	if len(results) != 1 || results[0].ID != entry.ID {
		t.Errorf("expected matching entry from search, got %+v", results)
	}

	// 4. Soft Delete
	if err := SoftDeleteEntry(nil, "gym", entry.ID, "logged by mistake", actor); err != nil {
		t.Fatalf("SoftDeleteEntry failed: %v", err)
	}

	// Normal query should not find deleted entry
	activeResults, err := QueryEntries(models.EntryFilter{
		PluginID: "gym",
	})
	if err != nil {
		t.Fatalf("QueryEntries failed: %v", err)
	}
	if len(activeResults) != 0 {
		t.Errorf("expected 0 active results after soft delete, got %d", len(activeResults))
	}

	// Query with status='all' should find it
	allResults, err := QueryEntries(models.EntryFilter{
		PluginID: "gym",
		Status:   "all",
	})
	if err != nil || len(allResults) != 1 {
		t.Errorf("expected 1 result with status=all, got count=%d", len(allResults))
	}

	// 5. Restore Entry
	if err := RestoreEntry(nil, "gym", entry.ID, 0, actor); err != nil {
		t.Fatalf("RestoreEntry failed: %v", err)
	}

	restored, err := GetEntry("gym", entry.ID)
	if err != nil {
		t.Fatalf("GetEntry after restore failed: %v", err)
	}
	if restored.DeletedAt != nil {
		t.Errorf("expected DeletedAt to be nil after restore")
	}
	if restored.Revision != 4 { // rev 1 (create) -> rev 2 (update) -> rev 3 (delete) -> rev 4 (restore)
		t.Errorf("expected revision 4 after restore, got %d", restored.Revision)
	}

	// 6. Time travel: Restore to revision 1 (weight: 70kg)
	historical, err := RestoreEntryToRevision("gym", "workout_set", entry.ID, 1, 4, actor)
	if err != nil {
		t.Fatalf("RestoreEntryToRevision failed: %v", err)
	}
	if historical.NumVal != 70.0 {
		t.Errorf("expected restored weight 70.0 from rev 1, got %f", historical.NumVal)
	}
	if historical.Revision != 5 {
		t.Errorf("expected revision 5 for new restoration state, got %d", historical.Revision)
	}
}

func TestEntryDateAndParentFiltering(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_filter.db")

	if err := InitDB(dbPath); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer CloseDB()
	_ = InitSchema()

	_ = RegisterPluginRecord(&models.PluginRecord{
		PluginID: "gym", Version: "1.0", APIVersion: "1", SchemaVersion: 1, Enabled: true, Source: "builtin", ManifestJSON: "{}",
	})

	sessionID := "sess_001"
	d1 := "2026-10-18"
	d2 := "2026-10-19"
	actor := models.DefaultUserActor()

	// Insert set 1 in session
	_ = CreateEntry(nil, &models.PluginEntry{
		PluginID: "gym", EntityType: "set", ParentID: &sessionID, EntryDate: &d1, NumVal: 80,
	}, actor)

	// Insert set 2 in session
	_ = CreateEntry(nil, &models.PluginEntry{
		PluginID: "gym", EntityType: "set", ParentID: &sessionID, EntryDate: &d1, NumVal: 85,
	}, actor)

	// Insert set on next day without parent
	_ = CreateEntry(nil, &models.PluginEntry{
		PluginID: "gym", EntityType: "set", EntryDate: &d2, NumVal: 90,
	}, actor)

	// Filter by parent_id
	sessSets, err := QueryEntries(models.EntryFilter{
		PluginID: "gym",
		ParentID: &sessionID,
	})
	if err != nil || len(sessSets) != 2 {
		t.Errorf("expected 2 sets for session, got count=%d, err=%v", len(sessSets), err)
	}

	// Filter by date
	daySets, err := QueryEntries(models.EntryFilter{
		PluginID:  "gym",
		EntryDate: &d1,
	})
	if err != nil || len(daySets) != 2 {
		t.Errorf("expected 2 sets for date %s, got count=%d, err=%v", d1, len(daySets), err)
	}
}
