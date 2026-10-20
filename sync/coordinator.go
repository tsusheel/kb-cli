package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
	"github.com/tsusheel/kb-cli/plugins/api"
)

const pgPluginSchemaSQL = `
CREATE TABLE IF NOT EXISTS plugins (
    plugin_id       TEXT PRIMARY KEY,
    version         TEXT NOT NULL,
    api_version     TEXT NOT NULL,
    schema_version  INTEGER NOT NULL DEFAULT 1,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    manifest_json   JSONB NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS plugin_entries (
    id              TEXT PRIMARY KEY,
    plugin_id       TEXT NOT NULL REFERENCES plugins(plugin_id),
    entity_type     TEXT NOT NULL,
    schema_version  INTEGER NOT NULL DEFAULT 1,
    parent_id       TEXT,
    status          TEXT NOT NULL DEFAULT 'active',
    occurred_at     TIMESTAMPTZ,
    entry_date      DATE,
    num_val         DOUBLE PRECISION DEFAULT 0,
    search_text     TEXT NOT NULL DEFAULT '',
    data            JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL,
    deleted_at      TIMESTAMPTZ,
    deleted_note    TEXT,
    revision        BIGINT NOT NULL DEFAULT 1,
    UNIQUE (plugin_id, id)
);

CREATE INDEX IF NOT EXISTS idx_pg_entries_date ON plugin_entries(plugin_id, entity_type, entry_date DESC);
CREATE INDEX IF NOT EXISTS idx_pg_entries_parent ON plugin_entries(plugin_id, parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_pg_entries_updated ON plugin_entries(updated_at, id);

CREATE TABLE IF NOT EXISTS entity_revisions (
    revision_id     TEXT PRIMARY KEY,
    plugin_id       TEXT NOT NULL,
    entity_type     TEXT NOT NULL,
    entity_id       TEXT NOT NULL,
    revision        BIGINT NOT NULL,
    schema_version  INTEGER NOT NULL,
    snapshot_json   JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    UNIQUE (plugin_id, entity_type, entity_id, revision)
);

CREATE TABLE IF NOT EXISTS sync_changes (
    cursor          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    change_id       TEXT NOT NULL UNIQUE,
    plugin_id       TEXT NOT NULL,
    entity_type     TEXT NOT NULL,
    entity_id       TEXT NOT NULL,
    operation       TEXT NOT NULL,
    entity_version  BIGINT NOT NULL,
    schema_version  INTEGER NOT NULL,
    payload_json    JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sync_changes_plugin_cursor ON sync_changes(plugin_id, cursor);
`

// SyncCoordinator manages the transactional outbox push and change-feed pull across plugins.
type SyncCoordinator struct {
	Client   *PostgresClient
	Registry *AdapterRegistry
}

func NewSyncCoordinator(client *PostgresClient, registry *AdapterRegistry) *SyncCoordinator {
	if registry == nil {
		registry = NewAdapterRegistry()
	}
	return &SyncCoordinator{
		Client:   client,
		Registry: registry,
	}
}

// EnsureRemoteSchema bootstraps all plugin cloud tables on PostgreSQL.
func (sc *SyncCoordinator) EnsureRemoteSchema(ctx context.Context) error {
	_, err := sc.Client.DB.ExecContext(ctx, pgPluginSchemaSQL)
	return err
}

// PushOutbox pushes pending local outbox changes to the remote PostgreSQL authority.
func (sc *SyncCoordinator) PushOutbox(ctx context.Context, batchSize int) (int, error) {
	if batchSize <= 0 {
		batchSize = 50
	}

	changes, err := db.GetPendingOutboxChanges(batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed fetching pending outbox changes: %w", err)
	}
	if len(changes) == 0 {
		return 0, nil
	}

	tx, err := sc.Client.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed starting remote transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var ackedIDs []string

	for _, c := range changes {
		adapter, ok := sc.Registry.Get(c.PluginID, c.EntityType)
		if ok {
			apiChange := api.Change{
				ChangeID:      c.ChangeID,
				PluginID:      c.PluginID,
				EntityType:    c.EntityType,
				EntityID:      c.EntityID,
				Operation:     c.Operation,
				BaseVersion:   c.BaseVersion,
				EntityVersion: c.EntityVersion,
				SchemaVersion: c.SchemaVersion,
				Payload:       []byte(c.PayloadJSON),
			}
			if valErr := adapter.Validate(ctx, apiChange); valErr != nil {
				_ = db.RecordOutboxFailure(c.ChangeID, valErr.Error())
				continue
			}
		}

		// Ensure parent plugin record exists in remote DB before inserting entry
		_, _ = tx.ExecContext(ctx, `INSERT INTO plugins (plugin_id, version, api_version, schema_version, enabled, manifest_json)
			VALUES ($1, '1.0.0', '1', 1, true, '{}')
			ON CONFLICT (plugin_id) DO NOTHING`, c.PluginID)

		if c.Operation == "upsert" {
			var entry models.PluginEntry
			if err := json.Unmarshal([]byte(c.PayloadJSON), &entry); err == nil {
				upsertQuery := `INSERT INTO plugin_entries (
					id, plugin_id, entity_type, schema_version, parent_id, status,
					occurred_at, entry_date, num_val, search_text, data,
					created_at, updated_at, deleted_at, deleted_note, revision
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
				ON CONFLICT (plugin_id, id) DO UPDATE SET
					status = EXCLUDED.status,
					num_val = EXCLUDED.num_val,
					search_text = EXCLUDED.search_text,
					data = EXCLUDED.data,
					updated_at = EXCLUDED.updated_at,
					deleted_at = EXCLUDED.deleted_at,
					deleted_note = EXCLUDED.deleted_note,
					revision = EXCLUDED.revision
				WHERE EXCLUDED.revision >= plugin_entries.revision`

				var parentIDVal sql.NullString
				if entry.ParentID != nil && *entry.ParentID != "" {
					parentIDVal = sql.NullString{String: *entry.ParentID, Valid: true}
				}

				var occurredAtVal sql.NullTime
				if entry.OccurredAt != nil && !entry.OccurredAt.IsZero() {
					occurredAtVal = sql.NullTime{Time: *entry.OccurredAt, Valid: true}
				}

				var entryDateVal sql.NullString
				if entry.EntryDate != nil && *entry.EntryDate != "" {
					entryDateVal = sql.NullString{String: *entry.EntryDate, Valid: true}
				}

				var deletedAtVal sql.NullTime
				if entry.DeletedAt != nil && !entry.DeletedAt.IsZero() {
					deletedAtVal = sql.NullTime{Time: *entry.DeletedAt, Valid: true}
				}

				var deletedNoteVal sql.NullString
				if entry.DeletedNote != nil && *entry.DeletedNote != "" {
					deletedNoteVal = sql.NullString{String: *entry.DeletedNote, Valid: true}
				}

				dataJSON := entry.Data
				if strings.TrimSpace(dataJSON) == "" {
					dataJSON = "{}"
				}

				_, err = tx.ExecContext(ctx, upsertQuery,
					entry.ID, entry.PluginID, entry.EntityType, entry.SchemaVersion, parentIDVal, entry.Status,
					occurredAtVal, entryDateVal, entry.NumVal, entry.SearchText, dataJSON,
					entry.CreatedAt, entry.UpdatedAt, deletedAtVal, deletedNoteVal, entry.Revision,
				)
				if err != nil {
					_ = db.RecordOutboxFailure(c.ChangeID, err.Error())
					return 0, fmt.Errorf("failed remote upsert for %s: %w", c.ChangeID, err)
				}
			}
		} else if c.Operation == "tombstone" {
			_, err = tx.ExecContext(ctx, `UPDATE plugin_entries SET deleted_at = NOW(), revision = $1 WHERE plugin_id = $2 AND id = $3 AND $1 >= revision`,
				c.EntityVersion, c.PluginID, c.EntityID)
			if err != nil {
				_ = db.RecordOutboxFailure(c.ChangeID, err.Error())
				return 0, fmt.Errorf("failed remote tombstone for %s: %w", c.ChangeID, err)
			}
		}

		// Insert into server change feed
		feedQuery := `INSERT INTO sync_changes (
			change_id, plugin_id, entity_type, entity_id, operation, entity_version, schema_version, payload_json
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (change_id) DO NOTHING`

		_, err = tx.ExecContext(ctx, feedQuery, c.ChangeID, c.PluginID, c.EntityType, c.EntityID, c.Operation, c.EntityVersion, c.SchemaVersion, c.PayloadJSON)
		if err != nil {
			_ = db.RecordOutboxFailure(c.ChangeID, err.Error())
			return 0, fmt.Errorf("failed inserting server change feed for %s: %w", c.ChangeID, err)
		}

		ackedIDs = append(ackedIDs, c.ChangeID)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed committing remote outbox batch: %w", err)
	}

	if err := db.AcknowledgeOutboxChanges(ackedIDs); err != nil {
		return len(ackedIDs), fmt.Errorf("remote batch committed but failed acknowledging local outbox: %w", err)
	}

	return len(ackedIDs), nil
}

// PullChangeFeed pulls server change-feed records after the local cursor and applies them locally.
func (sc *SyncCoordinator) PullChangeFeed(ctx context.Context, pageSize int) (int, error) {
	if pageSize <= 0 {
		pageSize = 100
	}

	cursorStr, err := db.GetSyncCursor("postgres", "sync_changes")
	if err != nil {
		return 0, fmt.Errorf("failed getting local cursor: %w", err)
	}

	var afterCursor int64 = 0
	if cursorStr != "" {
		if parsed, parseErr := strconv.ParseInt(cursorStr, 10, 64); parseErr == nil {
			afterCursor = parsed
		}
	}

	query := `SELECT cursor, change_id, plugin_id, entity_type, entity_id, operation, entity_version, schema_version, payload_json
		FROM sync_changes
		WHERE cursor > $1
		ORDER BY cursor ASC
		LIMIT $2`

	rows, err := sc.Client.DB.QueryContext(ctx, query, afterCursor, pageSize)
	if err != nil {
		return 0, fmt.Errorf("failed querying remote change feed: %w", err)
	}
	defer rows.Close()

	type feedItem struct {
		cursor        int64
		changeID      string
		pluginID      string
		entityType    string
		entityID      string
		operation     string
		entityVersion int64
		schemaVersion int
		payloadJSON   string
	}

	var items []feedItem
	for rows.Next() {
		var it feedItem
		if err := rows.Scan(&it.cursor, &it.changeID, &it.pluginID, &it.entityType, &it.entityID, &it.operation, &it.entityVersion, &it.schemaVersion, &it.payloadJSON); err != nil {
			return 0, fmt.Errorf("failed scanning change feed row: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	if len(items) == 0 {
		return 0, nil
	}

	// Apply items locally inside an atomic SQLite transaction
	tx, err := db.DB.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed starting local SQLite transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var lastAppliedCursor int64 = afterCursor

	for _, it := range items {
		// Ensure local plugin record exists
		_ = db.RegisterPluginRecord(&models.PluginRecord{
			PluginID:      it.pluginID,
			Version:       "1.0.0",
			APIVersion:    "1",
			SchemaVersion: 1,
			Enabled:       true,
			Source:        "remote_sync",
			ManifestJSON:  "{}",
		})

		if it.operation == "upsert" {
			var remoteEntry models.PluginEntry
			if err := json.Unmarshal([]byte(it.payloadJSON), &remoteEntry); err == nil {
				// Check existing local entry revision
				localEntry, getErr := db.GetEntry(it.pluginID, it.entityID)
				if getErr == db.ErrEntryNotFound {
					// Does not exist locally: insert directly
					_ = db.CreateEntry(tx, &remoteEntry, models.ActorContext{ActorType: "system", ActorID: "remote_sync"})
				} else if getErr == nil && remoteEntry.Revision >= localEntry.Revision {
					// Exists locally and remote revision is equal or newer: update
					_ = db.UpdateEntry(tx, &remoteEntry, localEntry.Revision, models.ActorContext{ActorType: "system", ActorID: "remote_sync"})
				}
			}
		} else if it.operation == "tombstone" {
			_ = db.SoftDeleteEntry(tx, it.pluginID, it.entityID, "deleted by remote replica", models.ActorContext{ActorType: "system", ActorID: "remote_sync"})
		}

		lastAppliedCursor = it.cursor
	}

	// Advance cursor
	if err := db.SetSyncCursor("postgres", "sync_changes", fmt.Sprintf("%d", lastAppliedCursor)); err != nil {
		return 0, fmt.Errorf("failed setting local sync cursor: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed committing local sync transaction: %w", err)
	}

	return len(items), nil
}
