package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tsusheel/kb-cli/models"
)

var (
	ErrEntryNotFound     = errors.New("plugin entry not found")
	ErrRevisionConflict  = errors.New("revision conflict: entity was modified concurrently")
	ErrAmbiguousEntryID  = errors.New("ambiguous short id, multiple entries matched")
	ErrRevisionNotFound  = errors.New("entity revision not found")
	ErrPluginNotFound    = errors.New("plugin not found")
)

// --- Plugin Registration & Capabilities ---

func RegisterPluginRecord(record *models.PluginRecord) error {
	if record.PluginID == "" {
		return fmt.Errorf("plugin_id cannot be empty")
	}
	now := time.Now().UTC()
	if record.InstalledAt.IsZero() {
		record.InstalledAt = now
	}
	record.UpdatedAt = now

	enabledInt := 0
	if record.Enabled {
		enabledInt = 1
	}

	query := `INSERT INTO plugins (plugin_id, version, api_version, schema_version, enabled, source, manifest_json, installed_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(plugin_id) DO UPDATE SET
			version = excluded.version,
			api_version = excluded.api_version,
			schema_version = excluded.schema_version,
			enabled = excluded.enabled,
			source = excluded.source,
			manifest_json = excluded.manifest_json,
			updated_at = excluded.updated_at`

	_, err := DB.Exec(query, record.PluginID, record.Version, record.APIVersion, record.SchemaVersion, enabledInt, record.Source, record.ManifestJSON, record.InstalledAt, record.UpdatedAt)
	return err
}

func GetPluginRecord(pluginID string) (*models.PluginRecord, error) {
	query := `SELECT plugin_id, version, api_version, schema_version, enabled, source, manifest_json, installed_at, updated_at
		FROM plugins WHERE plugin_id = ?`

	row := DB.QueryRow(query, pluginID)
	var r models.PluginRecord
	var enabledInt int
	if err := row.Scan(&r.PluginID, &r.Version, &r.APIVersion, &r.SchemaVersion, &enabledInt, &r.Source, &r.ManifestJSON, &r.InstalledAt, &r.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPluginNotFound
		}
		return nil, err
	}
	r.Enabled = (enabledInt == 1)
	return &r, nil
}

func ListPluginRecords() ([]models.PluginRecord, error) {
	query := `SELECT plugin_id, version, api_version, schema_version, enabled, source, manifest_json, installed_at, updated_at
		FROM plugins ORDER BY plugin_id ASC`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.PluginRecord
	for rows.Next() {
		var r models.PluginRecord
		var enabledInt int
		if err := rows.Scan(&r.PluginID, &r.Version, &r.APIVersion, &r.SchemaVersion, &enabledInt, &r.Source, &r.ManifestJSON, &r.InstalledAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		r.Enabled = (enabledInt == 1)
		records = append(records, r)
	}
	return records, rows.Err()
}

func SetPluginEnabled(pluginID string, enabled bool) error {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	res, err := DB.Exec(`UPDATE plugins SET enabled = ?, updated_at = ? WHERE plugin_id = ?`, enabledInt, time.Now().UTC(), pluginID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrPluginNotFound
	}
	return nil
}

func SetPluginCapability(pluginID, capability string, granted bool) error {
	grantedInt := 0
	if granted {
		grantedInt = 1
	}
	query := `INSERT INTO plugin_capabilities (plugin_id, capability, granted)
		VALUES (?, ?, ?)
		ON CONFLICT(plugin_id, capability) DO UPDATE SET granted = excluded.granted`
	_, err := DB.Exec(query, pluginID, capability, grantedInt)
	return err
}

func HasPluginCapability(pluginID, capability string) (bool, error) {
	var grantedInt int
	err := DB.QueryRow(`SELECT granted FROM plugin_capabilities WHERE plugin_id = ? AND capability = ?`, pluginID, capability).Scan(&grantedInt)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return grantedInt == 1, nil
}

func GetPluginCapabilities(pluginID string) ([]string, error) {
	rows, err := DB.Query(`SELECT capability FROM plugin_capabilities WHERE plugin_id = ? AND granted = 1`, pluginID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var caps []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		caps = append(caps, c)
	}
	return caps, rows.Err()
}

// --- Universal Plugin Entries CRUD ---

// ResolvePluginEntryID resolves full or short (prefix) IDs for a specific plugin.
func ResolvePluginEntryID(pluginID, idPrefix string) (string, error) {
	cleanID := strings.ReplaceAll(idPrefix, "-", "")
	if len(cleanID) == 32 {
		var exists string
		err := DB.QueryRow(`SELECT id FROM plugin_entries WHERE plugin_id = ? AND id = ?`, pluginID, cleanID).Scan(&exists)
		if err == sql.ErrNoRows {
			return "", ErrEntryNotFound
		} else if err != nil {
			return "", err
		}
		return cleanID, nil
	}

	query := `SELECT id FROM plugin_entries WHERE plugin_id = ? AND id LIKE ?`
	rows, err := DB.Query(query, pluginID, cleanID+"%")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var matchedID string
	count := 0
	for rows.Next() {
		count++
		if err := rows.Scan(&matchedID); err != nil {
			return "", err
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	if count == 0 {
		return "", ErrEntryNotFound
	}
	if count > 1 {
		return "", ErrAmbiguousEntryID
	}
	return matchedID, nil
}

// CreateEntry inserts a new universal entry, capturing revisions, audit events, and outbox changes.
func CreateEntry(txIn *sql.Tx, entry *models.PluginEntry, actor models.ActorContext) (err error) {
	if entry.PluginID == "" {
		return fmt.Errorf("entry PluginID is required")
	}
	if entry.EntityType == "" {
		return fmt.Errorf("entry EntityType is required")
	}

	if entry.ID == "" {
		entry.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
	} else {
		entry.ID = strings.ReplaceAll(entry.ID, "-", "")
	}

	now := time.Now().UTC()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now

	if entry.SchemaVersion <= 0 {
		entry.SchemaVersion = 1
	}
	if entry.Revision <= 0 {
		entry.Revision = 1
	}
	if entry.Status == "" {
		entry.Status = "active"
	}
	if entry.Data == "" {
		entry.Data = "{}"
	}

	tx := txIn
	if tx == nil {
		tx, err = DB.Begin()
		if err != nil {
			return fmt.Errorf("failed starting transaction: %w", err)
		}
		defer func() {
			if err != nil {
				_ = tx.Rollback()
			}
		}()
	}

	// 1. Insert into plugin_entries
	insertQuery := `INSERT INTO plugin_entries (
		id, plugin_id, entity_type, schema_version, parent_id, status,
		occurred_at, entry_date, num_val, search_text, data,
		created_at, updated_at, deleted_at, deleted_note, revision
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var occurredAtVal sql.NullTime
	if entry.OccurredAt != nil && !entry.OccurredAt.IsZero() {
		occurredAtVal = sql.NullTime{Time: *entry.OccurredAt, Valid: true}
	}

	var entryDateVal sql.NullString
	if entry.EntryDate != nil && *entry.EntryDate != "" {
		entryDateVal = sql.NullString{String: *entry.EntryDate, Valid: true}
	}

	var parentIDVal sql.NullString
	if entry.ParentID != nil && *entry.ParentID != "" {
		parentIDVal = sql.NullString{String: *entry.ParentID, Valid: true}
	}

	var deletedAtVal sql.NullTime
	if entry.DeletedAt != nil && !entry.DeletedAt.IsZero() {
		deletedAtVal = sql.NullTime{Time: *entry.DeletedAt, Valid: true}
	}

	var deletedNoteVal sql.NullString
	if entry.DeletedNote != nil && *entry.DeletedNote != "" {
		deletedNoteVal = sql.NullString{String: *entry.DeletedNote, Valid: true}
	}

	_, err = tx.Exec(insertQuery,
		entry.ID, entry.PluginID, entry.EntityType, entry.SchemaVersion, parentIDVal, entry.Status,
		occurredAtVal, entryDateVal, entry.NumVal, entry.SearchText, entry.Data,
		entry.CreatedAt, entry.UpdatedAt, deletedAtVal, deletedNoteVal, entry.Revision,
	)
	if err != nil {
		return fmt.Errorf("failed inserting plugin_entries: %w", err)
	}

	// 2. FTS5 Indexing
	if entry.SearchText != "" {
		_, err = tx.Exec(`INSERT INTO plugin_entries_fts (entry_id, plugin_id, entity_type, search_text) VALUES (?, ?, ?, ?)`,
			entry.ID, entry.PluginID, entry.EntityType, entry.SearchText)
		if err != nil {
			return fmt.Errorf("failed indexing search text: %w", err)
		}
	}

	// 3. Entity Revision Snapshot
	snapshotBytes, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed encoding snapshot: %w", err)
	}
	snapshotJSON := string(snapshotBytes)
	revID := strings.ReplaceAll(uuid.New().String(), "-", "")

	_, err = tx.Exec(`INSERT INTO entity_revisions (revision_id, plugin_id, entity_type, entity_id, revision, schema_version, snapshot_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		revID, entry.PluginID, entry.EntityType, entry.ID, entry.Revision, entry.SchemaVersion, snapshotJSON, now)
	if err != nil {
		return fmt.Errorf("failed recording revision: %w", err)
	}

	// 4. Audit Log
	auditID := strings.ReplaceAll(uuid.New().String(), "-", "")
	actorType := actor.ActorType
	if actorType == "" {
		actorType = "user"
	}
	summary := fmt.Sprintf("created %s", entry.EntityType)

	_, err = tx.Exec(`INSERT INTO audit_logs (
		id, plugin_id, entity_type, entity_id, revision, action, actor_type, actor_id, request_id,
		changes_summary, snapshot_json, diff_json, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		auditID, entry.PluginID, entry.EntityType, entry.ID, entry.Revision, string(models.ActionCreated), actorType,
		actor.ActorID, actor.RequestID, summary, snapshotJSON, "{}", now)
	if err != nil {
		return fmt.Errorf("failed recording audit log: %w", err)
	}

	// 5. Transactional Outbox
	changeID := strings.ReplaceAll(uuid.New().String(), "-", "")
	_, err = tx.Exec(`INSERT INTO sync_outbox (
		change_id, plugin_id, entity_type, entity_id, operation, base_version, entity_version, schema_version, payload_json, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		changeID, entry.PluginID, entry.EntityType, entry.ID, "upsert", 0, entry.Revision, entry.SchemaVersion, snapshotJSON, now)
	if err != nil {
		return fmt.Errorf("failed enqueueing outbox change: %w", err)
	}

	if txIn == nil {
		return tx.Commit()
	}
	return nil
}

// GetEntry retrieves a universal entry by pluginID and ID (or short prefix).
func GetEntry(pluginID, id string) (*models.PluginEntry, error) {
	fullID, err := ResolvePluginEntryID(pluginID, id)
	if err != nil {
		return nil, err
	}

	query := `SELECT id, plugin_id, entity_type, schema_version, parent_id, status,
		occurred_at, entry_date, num_val, search_text, data,
		created_at, updated_at, deleted_at, deleted_note, revision
		FROM plugin_entries WHERE plugin_id = ? AND id = ?`

	row := DB.QueryRow(query, pluginID, fullID)
	return scanEntry(row)
}

func scanEntry(scanner interface {
	Scan(dest ...any) error
}) (*models.PluginEntry, error) {
	var e models.PluginEntry
	var parentID sql.NullString
	var occurredAt sql.NullTime
	var entryDate sql.NullString
	var deletedAt sql.NullTime
	var deletedNote sql.NullString

	err := scanner.Scan(
		&e.ID, &e.PluginID, &e.EntityType, &e.SchemaVersion, &parentID, &e.Status,
		&occurredAt, &entryDate, &e.NumVal, &e.SearchText, &e.Data,
		&e.CreatedAt, &e.UpdatedAt, &deletedAt, &deletedNote, &e.Revision,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrEntryNotFound
		}
		return nil, err
	}

	if parentID.Valid {
		e.ParentID = &parentID.String
	}
	if occurredAt.Valid {
		e.OccurredAt = &occurredAt.Time
	}
	if entryDate.Valid {
		e.EntryDate = &entryDate.String
	}
	if deletedAt.Valid {
		e.DeletedAt = &deletedAt.Time
	}
	if deletedNote.Valid {
		e.DeletedNote = &deletedNote.String
	}

	return &e, nil
}

// UpdateEntry updates an existing entry with optimistic concurrency verification.
func UpdateEntry(txIn *sql.Tx, entry *models.PluginEntry, expectedRevision int64, actor models.ActorContext) (err error) {
	if entry.PluginID == "" || entry.ID == "" {
		return fmt.Errorf("plugin_id and id are required")
	}

	fullID, err := ResolvePluginEntryID(entry.PluginID, entry.ID)
	if err != nil {
		return err
	}
	entry.ID = fullID

	tx := txIn
	if tx == nil {
		tx, err = DB.Begin()
		if err != nil {
			return fmt.Errorf("failed starting transaction: %w", err)
		}
		defer func() {
			if err != nil {
				_ = tx.Rollback()
			}
		}()
	}

	// 1. Fetch current revision
	var currentRevision int64
	var oldSearchText string
	err = tx.QueryRow(`SELECT revision, search_text FROM plugin_entries WHERE plugin_id = ? AND id = ?`, entry.PluginID, entry.ID).Scan(&currentRevision, &oldSearchText)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrEntryNotFound
		}
		return err
	}

	if expectedRevision > 0 && currentRevision != expectedRevision {
		return fmt.Errorf("%w: expected revision %d, current is %d", ErrRevisionConflict, expectedRevision, currentRevision)
	}

	now := time.Now().UTC()
	entry.Revision = currentRevision + 1
	entry.UpdatedAt = now

	// 2. Execute update
	updateQuery := `UPDATE plugin_entries SET
		entity_type = ?, schema_version = ?, parent_id = ?, status = ?,
		occurred_at = ?, entry_date = ?, num_val = ?, search_text = ?, data = ?,
		updated_at = ?, revision = ?
		WHERE plugin_id = ? AND id = ?`

	var occurredAtVal sql.NullTime
	if entry.OccurredAt != nil && !entry.OccurredAt.IsZero() {
		occurredAtVal = sql.NullTime{Time: *entry.OccurredAt, Valid: true}
	}
	var entryDateVal sql.NullString
	if entry.EntryDate != nil && *entry.EntryDate != "" {
		entryDateVal = sql.NullString{String: *entry.EntryDate, Valid: true}
	}
	var parentIDVal sql.NullString
	if entry.ParentID != nil && *entry.ParentID != "" {
		parentIDVal = sql.NullString{String: *entry.ParentID, Valid: true}
	}

	_, err = tx.Exec(updateQuery,
		entry.EntityType, entry.SchemaVersion, parentIDVal, entry.Status,
		occurredAtVal, entryDateVal, entry.NumVal, entry.SearchText, entry.Data,
		entry.UpdatedAt, entry.Revision, entry.PluginID, entry.ID,
	)
	if err != nil {
		return fmt.Errorf("failed updating plugin_entries: %w", err)
	}

	// 3. Synchronize FTS5
	if oldSearchText != entry.SearchText {
		_, _ = tx.Exec(`DELETE FROM plugin_entries_fts WHERE entry_id = ?`, entry.ID)
		if entry.SearchText != "" {
			_, err = tx.Exec(`INSERT INTO plugin_entries_fts (entry_id, plugin_id, entity_type, search_text) VALUES (?, ?, ?, ?)`,
				entry.ID, entry.PluginID, entry.EntityType, entry.SearchText)
			if err != nil {
				return fmt.Errorf("failed updating FTS: %w", err)
			}
		}
	}

	// 4. Snapshot
	snapshotBytes, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed encoding snapshot: %w", err)
	}
	snapshotJSON := string(snapshotBytes)
	revID := strings.ReplaceAll(uuid.New().String(), "-", "")

	_, err = tx.Exec(`INSERT INTO entity_revisions (revision_id, plugin_id, entity_type, entity_id, revision, schema_version, snapshot_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		revID, entry.PluginID, entry.EntityType, entry.ID, entry.Revision, entry.SchemaVersion, snapshotJSON, now)
	if err != nil {
		return fmt.Errorf("failed recording revision: %w", err)
	}

	// 5. Audit Log
	auditID := strings.ReplaceAll(uuid.New().String(), "-", "")
	actorType := actor.ActorType
	if actorType == "" {
		actorType = "user"
	}
	summary := fmt.Sprintf("updated %s (rev %d -> %d)", entry.EntityType, currentRevision, entry.Revision)

	_, err = tx.Exec(`INSERT INTO audit_logs (
		id, plugin_id, entity_type, entity_id, revision, action, actor_type, actor_id, request_id,
		before_revision, after_revision, changes_summary, snapshot_json, diff_json, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		auditID, entry.PluginID, entry.EntityType, entry.ID, entry.Revision, string(models.ActionUpdated), actorType,
		actor.ActorID, actor.RequestID, fmt.Sprintf("%d", currentRevision), fmt.Sprintf("%d", entry.Revision),
		summary, snapshotJSON, "{}", now)
	if err != nil {
		return fmt.Errorf("failed recording audit log: %w", err)
	}

	// 6. Outbox
	changeID := strings.ReplaceAll(uuid.New().String(), "-", "")
	_, err = tx.Exec(`INSERT INTO sync_outbox (
		change_id, plugin_id, entity_type, entity_id, operation, base_version, entity_version, schema_version, payload_json, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		changeID, entry.PluginID, entry.EntityType, entry.ID, "upsert", currentRevision, entry.Revision, entry.SchemaVersion, snapshotJSON, now)
	if err != nil {
		return fmt.Errorf("failed enqueueing outbox change: %w", err)
	}

	if txIn == nil {
		return tx.Commit()
	}
	return nil
}

// SoftDeleteEntry marks an entry as soft-deleted and enqueues a tombstone outbox change.
func SoftDeleteEntry(txIn *sql.Tx, pluginID, id, reason string, actor models.ActorContext) (err error) {
	fullID, err := ResolvePluginEntryID(pluginID, id)
	if err != nil {
		return err
	}

	entry, err := GetEntry(pluginID, fullID)
	if err != nil {
		return err
	}

	tx := txIn
	if tx == nil {
		tx, err = DB.Begin()
		if err != nil {
			return fmt.Errorf("failed starting transaction: %w", err)
		}
		defer func() {
			if err != nil {
				_ = tx.Rollback()
			}
		}()
	}

	now := time.Now().UTC()
	currentRevision := entry.Revision
	entry.Revision = currentRevision + 1
	entry.DeletedAt = &now
	entry.DeletedNote = &reason
	entry.UpdatedAt = now

	_, err = tx.Exec(`UPDATE plugin_entries SET deleted_at = ?, deleted_note = ?, updated_at = ?, revision = ?
		WHERE plugin_id = ? AND id = ?`,
		now, reason, now, entry.Revision, pluginID, fullID)
	if err != nil {
		return fmt.Errorf("failed soft-deleting entry: %w", err)
	}

	// Remove from FTS5
	_, _ = tx.Exec(`DELETE FROM plugin_entries_fts WHERE entry_id = ?`, fullID)

	// Snapshot
	snapshotBytes, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed encoding snapshot: %w", err)
	}
	snapshotJSON := string(snapshotBytes)
	revID := strings.ReplaceAll(uuid.New().String(), "-", "")

	_, err = tx.Exec(`INSERT INTO entity_revisions (revision_id, plugin_id, entity_type, entity_id, revision, schema_version, snapshot_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		revID, entry.PluginID, entry.EntityType, entry.ID, entry.Revision, entry.SchemaVersion, snapshotJSON, now)
	if err != nil {
		return fmt.Errorf("failed recording revision: %w", err)
	}

	// Audit Log
	auditID := strings.ReplaceAll(uuid.New().String(), "-", "")
	actorType := actor.ActorType
	if actorType == "" {
		actorType = "user"
	}
	summary := fmt.Sprintf("deleted %s: %s", entry.EntityType, reason)

	_, err = tx.Exec(`INSERT INTO audit_logs (
		id, plugin_id, entity_type, entity_id, revision, action, actor_type, actor_id, request_id,
		before_revision, after_revision, changes_summary, snapshot_json, diff_json, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		auditID, entry.PluginID, entry.EntityType, entry.ID, entry.Revision, string(models.ActionDeleted), actorType,
		actor.ActorID, actor.RequestID, fmt.Sprintf("%d", currentRevision), fmt.Sprintf("%d", entry.Revision),
		summary, snapshotJSON, "{}", now)
	if err != nil {
		return fmt.Errorf("failed recording audit log: %w", err)
	}

	// Outbox tombstone
	changeID := strings.ReplaceAll(uuid.New().String(), "-", "")
	_, err = tx.Exec(`INSERT INTO sync_outbox (
		change_id, plugin_id, entity_type, entity_id, operation, base_version, entity_version, schema_version, payload_json, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		changeID, entry.PluginID, entry.EntityType, entry.ID, "tombstone", currentRevision, entry.Revision, entry.SchemaVersion, snapshotJSON, now)
	if err != nil {
		return fmt.Errorf("failed enqueueing outbox tombstone: %w", err)
	}

	if txIn == nil {
		return tx.Commit()
	}
	return nil
}

// RestoreEntry restores a soft-deleted entry.
func RestoreEntry(txIn *sql.Tx, pluginID, id string, expectedRevision int64, actor models.ActorContext) (err error) {
	fullID, err := ResolvePluginEntryID(pluginID, id)
	if err != nil {
		return err
	}

	entry, err := GetEntry(pluginID, fullID)
	if err != nil {
		return err
	}

	if expectedRevision > 0 && entry.Revision != expectedRevision {
		return fmt.Errorf("%w: expected revision %d, current is %d", ErrRevisionConflict, expectedRevision, entry.Revision)
	}

	tx := txIn
	if tx == nil {
		tx, err = DB.Begin()
		if err != nil {
			return fmt.Errorf("failed starting transaction: %w", err)
		}
		defer func() {
			if err != nil {
				_ = tx.Rollback()
			}
		}()
	}

	now := time.Now().UTC()
	currentRevision := entry.Revision
	entry.Revision = currentRevision + 1
	entry.DeletedAt = nil
	entry.DeletedNote = nil
	entry.UpdatedAt = now

	_, err = tx.Exec(`UPDATE plugin_entries SET deleted_at = NULL, deleted_note = NULL, updated_at = ?, revision = ?
		WHERE plugin_id = ? AND id = ?`,
		now, entry.Revision, pluginID, fullID)
	if err != nil {
		return fmt.Errorf("failed restoring entry: %w", err)
	}

	// Re-insert into FTS5
	if entry.SearchText != "" {
		_, _ = tx.Exec(`DELETE FROM plugin_entries_fts WHERE entry_id = ?`, fullID)
		_, _ = tx.Exec(`INSERT INTO plugin_entries_fts (entry_id, plugin_id, entity_type, search_text) VALUES (?, ?, ?, ?)`,
			entry.ID, entry.PluginID, entry.EntityType, entry.SearchText)
	}

	// Snapshot
	snapshotBytes, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed encoding snapshot: %w", err)
	}
	snapshotJSON := string(snapshotBytes)
	revID := strings.ReplaceAll(uuid.New().String(), "-", "")

	_, err = tx.Exec(`INSERT INTO entity_revisions (revision_id, plugin_id, entity_type, entity_id, revision, schema_version, snapshot_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		revID, entry.PluginID, entry.EntityType, entry.ID, entry.Revision, entry.SchemaVersion, snapshotJSON, now)
	if err != nil {
		return fmt.Errorf("failed recording revision: %w", err)
	}

	// Audit Log
	auditID := strings.ReplaceAll(uuid.New().String(), "-", "")
	actorType := actor.ActorType
	if actorType == "" {
		actorType = "user"
	}
	summary := fmt.Sprintf("restored %s", entry.EntityType)

	_, err = tx.Exec(`INSERT INTO audit_logs (
		id, plugin_id, entity_type, entity_id, revision, action, actor_type, actor_id, request_id,
		before_revision, after_revision, changes_summary, snapshot_json, diff_json, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		auditID, entry.PluginID, entry.EntityType, entry.ID, entry.Revision, string(models.ActionRestored), actorType,
		actor.ActorID, actor.RequestID, fmt.Sprintf("%d", currentRevision), fmt.Sprintf("%d", entry.Revision),
		summary, snapshotJSON, "{}", now)
	if err != nil {
		return fmt.Errorf("failed recording audit log: %w", err)
	}

	// Outbox
	changeID := strings.ReplaceAll(uuid.New().String(), "-", "")
	_, err = tx.Exec(`INSERT INTO sync_outbox (
		change_id, plugin_id, entity_type, entity_id, operation, base_version, entity_version, schema_version, payload_json, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		changeID, entry.PluginID, entry.EntityType, entry.ID, "upsert", currentRevision, entry.Revision, entry.SchemaVersion, snapshotJSON, now)
	if err != nil {
		return fmt.Errorf("failed enqueueing outbox change: %w", err)
	}

	if txIn == nil {
		return tx.Commit()
	}
	return nil
}

// QueryEntries executes multi-faceted queries across universal plugin entries.
func QueryEntries(filter models.EntryFilter) ([]models.PluginEntry, error) {
	if filter.PluginID == "" {
		return nil, fmt.Errorf("filter PluginID is required")
	}

	var conditions []string
	var args []any

	conditions = append(conditions, "plugin_entries.plugin_id = ?")
	args = append(args, filter.PluginID)

	if filter.EntityType != "" {
		conditions = append(conditions, "plugin_entries.entity_type = ?")
		args = append(args, filter.EntityType)
	}

	if filter.ParentID != nil {
		if *filter.ParentID == "" {
			conditions = append(conditions, "plugin_entries.parent_id IS NULL")
		} else {
			conditions = append(conditions, "plugin_entries.parent_id = ?")
			args = append(args, *filter.ParentID)
		}
	}

	if filter.Status != "" && filter.Status != "all" && filter.Status != "deleted" {
		conditions = append(conditions, "plugin_entries.status = ?")
		args = append(args, filter.Status)
	} else if filter.Status == "deleted" {
		conditions = append(conditions, "plugin_entries.deleted_at IS NOT NULL")
	}

	if filter.EntryDate != nil && *filter.EntryDate != "" {
		conditions = append(conditions, "plugin_entries.entry_date = ?")
		args = append(args, *filter.EntryDate)
	}

	if filter.FromDate != nil && !filter.FromDate.IsZero() {
		conditions = append(conditions, "(plugin_entries.occurred_at >= ? OR plugin_entries.entry_date >= ?)")
		args = append(args, *filter.FromDate, filter.FromDate.Format("2006-01-02"))
	}

	if filter.ToDate != nil && !filter.ToDate.IsZero() {
		conditions = append(conditions, "(plugin_entries.occurred_at <= ? OR plugin_entries.entry_date <= ?)")
		args = append(args, *filter.ToDate, filter.ToDate.Format("2006-01-02"))
	}

	// Always exclude deleted items unless explicitly requested (e.g., status == 'all' or 'deleted')
	if filter.Status != "all" && filter.Status != "deleted" {
		conditions = append(conditions, "plugin_entries.deleted_at IS NULL")
	}

	var joinFTS string
	if filter.SearchText != "" {
		joinFTS = ` JOIN plugin_entries_fts ON plugin_entries.id = plugin_entries_fts.entry_id `
		conditions = append(conditions, "plugin_entries_fts.search_text MATCH ?")
		args = append(args, filter.SearchText)
	}

	whereClause := strings.Join(conditions, " AND ")

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	query := fmt.Sprintf(`SELECT plugin_entries.id, plugin_entries.plugin_id, plugin_entries.entity_type,
		plugin_entries.schema_version, plugin_entries.parent_id, plugin_entries.status,
		plugin_entries.occurred_at, plugin_entries.entry_date, plugin_entries.num_val,
		plugin_entries.search_text, plugin_entries.data, plugin_entries.created_at,
		plugin_entries.updated_at, plugin_entries.deleted_at, plugin_entries.deleted_note,
		plugin_entries.revision
		FROM plugin_entries%s
		WHERE %s
		ORDER BY COALESCE(plugin_entries.occurred_at, plugin_entries.created_at) DESC, plugin_entries.entry_date DESC
		LIMIT ? OFFSET ?`, joinFTS, whereClause)

	args = append(args, limit, filter.Offset)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query entries failed: %w", err)
	}
	defer rows.Close()

	var entries []models.PluginEntry
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, *entry)
	}

	return entries, rows.Err()
}

// GetEntityRevisions retrieves point-in-time snapshots for an entity.
func GetEntityRevisions(pluginID, entityType, entityID string, limit int) ([]models.EntityRevision, error) {
	if limit <= 0 {
		limit = 30
	}

	query := `SELECT revision_id, plugin_id, entity_type, entity_id, revision, schema_version, snapshot_json, created_at
		FROM entity_revisions
		WHERE plugin_id = ? AND entity_type = ? AND entity_id = ?
		ORDER BY revision DESC
		LIMIT ?`

	rows, err := DB.Query(query, pluginID, entityType, entityID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revs []models.EntityRevision
	for rows.Next() {
		var r models.EntityRevision
		if err := rows.Scan(&r.RevisionID, &r.PluginID, &r.EntityType, &r.EntityID, &r.Revision, &r.SchemaVersion, &r.SnapshotJSON, &r.CreatedAt); err != nil {
			return nil, err
		}
		revs = append(revs, r)
	}
	return revs, rows.Err()
}

// RestoreEntryToRevision restores an entity to a past revision snapshot as a new revision.
func RestoreEntryToRevision(pluginID, entityType, entityID string, targetRevision int64, expectedCurrentRevision int64, actor models.ActorContext) (*models.PluginEntry, error) {
	fullID, err := ResolvePluginEntryID(pluginID, entityID)
	if err != nil {
		return nil, err
	}

	var snapshotJSON string
	err = DB.QueryRow(`SELECT snapshot_json FROM entity_revisions
		WHERE plugin_id = ? AND entity_type = ? AND entity_id = ? AND revision = ?`,
		pluginID, entityType, fullID, targetRevision).Scan(&snapshotJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRevisionNotFound
		}
		return nil, err
	}

	var restored models.PluginEntry
	if err := json.Unmarshal([]byte(snapshotJSON), &restored); err != nil {
		return nil, fmt.Errorf("failed unmarshaling snapshot JSON: %w", err)
	}

	// Update the entry state to match snapshot
	if err := UpdateEntry(nil, &restored, expectedCurrentRevision, actor); err != nil {
		return nil, err
	}

	return &restored, nil
}
