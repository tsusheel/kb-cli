package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type Migration struct {
	Version int
	Name    string
	Run     func(tx *sql.Tx) error
}

type MigrationRecord struct {
	Version   int       `json:"version"`
	Name      string    `json:"name"`
	AppliedAt time.Time `json:"applied_at"`
}

type PluginMigrationRecord struct {
	PluginID  string    `json:"plugin_id"`
	Version   int       `json:"version"`
	Checksum  string    `json:"checksum"`
	AppliedAt time.Time `json:"applied_at"`
}

// EnsureSchemaMigrationsTable guarantees that the tracking table exists.
func EnsureSchemaMigrationsTable(targetDB *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS schema_migrations (
		version     INTEGER PRIMARY KEY,
		name        TEXT NOT NULL,
		applied_at  DATETIME NOT NULL
	);`
	_, err := targetDB.Exec(query)
	return err
}

// RunMigrations executes all pending core schema migrations in sequence.
func RunMigrations(targetDB *sql.DB) error {
	if targetDB == nil {
		targetDB = DB
	}
	if err := EnsureSchemaMigrationsTable(targetDB); err != nil {
		return fmt.Errorf("failed creating schema_migrations: %w", err)
	}

	// Bootstrap legacy check: If 'notes' table already exists, mark migration 1 as applied
	var notesTableExists int
	err := targetDB.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='notes'`).Scan(&notesTableExists)
	if err == nil && notesTableExists > 0 {
		var v1Exists int
		_ = targetDB.QueryRow(`SELECT count(*) FROM schema_migrations WHERE version=1`).Scan(&v1Exists)
		if v1Exists == 0 {
			_, _ = targetDB.Exec(`INSERT INTO schema_migrations (version, name, applied_at) VALUES (1, 'initial_core_schema', ?)`, time.Now())
		}
	}

	migrations := []Migration{
		{
			Version: 1,
			Name:    "initial_core_schema",
			Run: func(tx *sql.Tx) error {
				_, err := tx.Exec(schemaSQL)
				return err
			},
		},
		{
			Version: 2,
			Name:    "multi_plugin_platform_v1",
			Run:     migrationV2PluginPlatform,
		},
		{
			Version: 3,
			Name:    "flexible_docs_tree_v1",
			Run:     migrationV3DocsTree,
		},
		{
			Version: 4,
			Name:    "flexible_docs_tags_v1",
			Run:     migrationV4DocTags,
		},
	}

	for _, m := range migrations {
		var count int
		err := targetDB.QueryRow(`SELECT count(*) FROM schema_migrations WHERE version = ?`, m.Version).Scan(&count)
		if err != nil {
			return fmt.Errorf("error checking migration version %d: %w", m.Version, err)
		}
		if count > 0 {
			continue // already applied
		}

		tx, err := targetDB.Begin()
		if err != nil {
			return fmt.Errorf("failed starting transaction for migration %d (%s): %w", m.Version, m.Name, err)
		}

		if err := m.Run(tx); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed executing migration %d (%s): %w", m.Version, m.Name, err)
		}

		now := time.Now()
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`, m.Version, m.Name, now); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed recording migration %d (%s): %w", m.Version, m.Name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed committing migration %d (%s): %w", m.Version, m.Name, err)
		}
	}

	return nil
}

func migrationV2PluginPlatform(tx *sql.Tx) error {
	queries := []string{
		// 1. Plugin registry and migrations
		`CREATE TABLE IF NOT EXISTS plugins (
			plugin_id       TEXT PRIMARY KEY,
			version         TEXT NOT NULL,
			api_version     TEXT NOT NULL,
			schema_version  INTEGER NOT NULL DEFAULT 1,
			enabled         INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
			source          TEXT NOT NULL,
			manifest_json   TEXT NOT NULL,
			installed_at    DATETIME NOT NULL,
			updated_at      DATETIME NOT NULL
		);`,

		`CREATE TABLE IF NOT EXISTS plugin_capabilities (
			plugin_id       TEXT NOT NULL,
			capability      TEXT NOT NULL,
			granted         INTEGER NOT NULL DEFAULT 0 CHECK (granted IN (0, 1)),
			PRIMARY KEY (plugin_id, capability)
		);`,

		`CREATE TABLE IF NOT EXISTS plugin_migrations (
			plugin_id       TEXT NOT NULL,
			version         INTEGER NOT NULL,
			checksum        TEXT NOT NULL,
			applied_at      DATETIME NOT NULL,
			PRIMARY KEY (plugin_id, version)
		);`,

		// 2. Universal plugin entries
		`CREATE TABLE IF NOT EXISTS plugin_entries (
			id              TEXT PRIMARY KEY,
			plugin_id       TEXT NOT NULL,
			entity_type     TEXT NOT NULL,
			schema_version  INTEGER NOT NULL DEFAULT 1,
			parent_id       TEXT,
			status          TEXT NOT NULL DEFAULT 'active',
			occurred_at     DATETIME,
			entry_date      TEXT,
			num_val         REAL DEFAULT 0,
			search_text     TEXT NOT NULL DEFAULT '',
			data            TEXT NOT NULL DEFAULT '{}',
			created_at      DATETIME NOT NULL,
			updated_at      DATETIME NOT NULL,
			deleted_at      DATETIME,
			deleted_note    TEXT,
			revision        INTEGER NOT NULL DEFAULT 1,
			UNIQUE (plugin_id, id),
			FOREIGN KEY (plugin_id) REFERENCES plugins(plugin_id)
		);`,

		`CREATE INDEX IF NOT EXISTS idx_plugin_entries_date ON plugin_entries(plugin_id, entity_type, entry_date);`,
		`CREATE INDEX IF NOT EXISTS idx_plugin_entries_parent ON plugin_entries(plugin_id, parent_id) WHERE parent_id IS NOT NULL;`,
		`CREATE INDEX IF NOT EXISTS idx_plugin_entries_updated ON plugin_entries(updated_at, id);`,
		`CREATE INDEX IF NOT EXISTS idx_plugin_entries_status ON plugin_entries(plugin_id, status, entry_date);`,
		`CREATE INDEX IF NOT EXISTS idx_plugin_entries_numeric ON plugin_entries(plugin_id, entity_type, num_val);`,

		// 3. FTS5 Search for universal entries
		`CREATE VIRTUAL TABLE IF NOT EXISTS plugin_entries_fts USING fts5(
			entry_id UNINDEXED,
			plugin_id UNINDEXED,
			entity_type UNINDEXED,
			search_text,
			tokenize = 'unicode61'
		);`,

		// 4. Entity Revisions
		`CREATE TABLE IF NOT EXISTS entity_revisions (
			revision_id     TEXT PRIMARY KEY,
			plugin_id       TEXT NOT NULL,
			entity_type     TEXT NOT NULL,
			entity_id       TEXT NOT NULL,
			revision        INTEGER NOT NULL,
			schema_version  INTEGER NOT NULL,
			snapshot_json   TEXT NOT NULL,
			created_at      DATETIME NOT NULL,
			UNIQUE (plugin_id, entity_type, entity_id, revision)
		);`,

		`CREATE INDEX IF NOT EXISTS idx_revisions_entity ON entity_revisions(plugin_id, entity_type, entity_id, revision DESC);`,

		// 5. Cross-Entity Tags & Links
		`CREATE TABLE IF NOT EXISTS entity_tags (
			plugin_id       TEXT NOT NULL,
			entity_type     TEXT NOT NULL,
			entity_id       TEXT NOT NULL,
			tag_id          TEXT NOT NULL,
			PRIMARY KEY (plugin_id, entity_type, entity_id, tag_id),
			FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS entity_links (
			link_id         TEXT PRIMARY KEY,
			source_plugin   TEXT NOT NULL,
			source_type     TEXT NOT NULL,
			source_id       TEXT NOT NULL,
			target_plugin   TEXT NOT NULL,
			target_type     TEXT NOT NULL,
			target_id       TEXT NOT NULL,
			relation_type   TEXT NOT NULL DEFAULT 'related_to',
			created_at      DATETIME NOT NULL,
			UNIQUE (source_plugin, source_type, source_id, target_plugin, target_type, target_id, relation_type)
		);`,

		// 6. Transactional Outbox & Sync Cursors
		`CREATE TABLE IF NOT EXISTS sync_outbox (
			sequence        INTEGER PRIMARY KEY AUTOINCREMENT,
			change_id       TEXT NOT NULL UNIQUE,
			plugin_id       TEXT NOT NULL,
			entity_type     TEXT NOT NULL,
			entity_id       TEXT NOT NULL,
			operation       TEXT NOT NULL CHECK (operation IN ('upsert', 'tombstone')),
			base_version    INTEGER NOT NULL DEFAULT 0,
			entity_version  INTEGER NOT NULL,
			schema_version  INTEGER NOT NULL,
			payload_json    TEXT NOT NULL,
			created_at      DATETIME NOT NULL,
			attempts        INTEGER NOT NULL DEFAULT 0,
			last_error      TEXT,
			acknowledged_at DATETIME
		);`,

		`CREATE INDEX IF NOT EXISTS idx_outbox_pending ON sync_outbox(sequence) WHERE acknowledged_at IS NULL;`,

		`CREATE TABLE IF NOT EXISTS sync_cursors (
			remote_id       TEXT NOT NULL,
			stream_id       TEXT NOT NULL,
			cursor_value    TEXT,
			updated_at      DATETIME NOT NULL,
			PRIMARY KEY (remote_id, stream_id)
		);`,
	}

	for _, q := range queries {
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("failed executing DDL statement [%s]: %w", q, err)
		}
	}

	// 7. Backward-compatible column extensions for audit_logs
	auditColumns := map[string]string{
		"plugin_id":       "TEXT DEFAULT 'core'",
		"revision":        "INTEGER DEFAULT 1",
		"actor_type":      "TEXT DEFAULT 'user'",
		"actor_id":        "TEXT",
		"request_id":      "TEXT",
		"before_revision": "TEXT",
		"after_revision":  "TEXT",
		"diff_json":       "TEXT DEFAULT '{}'",
	}

	existingCols, err := getTableColumns(tx, "audit_logs")
	if err != nil {
		return fmt.Errorf("failed inspecting audit_logs columns: %w", err)
	}

	for colName, colDef := range auditColumns {
		if !existingCols[colName] {
			alterQuery := fmt.Sprintf("ALTER TABLE audit_logs ADD COLUMN %s %s;", colName, colDef)
			if _, err := tx.Exec(alterQuery); err != nil {
				return fmt.Errorf("failed adding column %s to audit_logs: %w", colName, err)
			}
		}
	}

	_, _ = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_plugin_entity ON audit_logs(plugin_id, entity_type, entity_id, created_at DESC);`)

	return nil
}

func migrationV3DocsTree(tx *sql.Tx) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS doc_nodes (
			id            TEXT PRIMARY KEY,
			parent_id     TEXT,
			type          TEXT NOT NULL CHECK (type IN ('folder', 'doc')),
			name          TEXT NOT NULL,
			title         TEXT NOT NULL,
			slug          TEXT NOT NULL,
			path          TEXT NOT NULL UNIQUE,
			content       TEXT,
			content_type  TEXT NOT NULL DEFAULT 'text/markdown',
			order_index   INTEGER NOT NULL DEFAULT 0,
			created_at    DATETIME NOT NULL,
			updated_at    DATETIME NOT NULL,
			deleted_at    DATETIME,
			deleted_note  TEXT,
			FOREIGN KEY (parent_id) REFERENCES doc_nodes(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_doc_nodes_parent ON doc_nodes(parent_id, order_index);`,
		`CREATE INDEX IF NOT EXISTS idx_doc_nodes_path ON doc_nodes(path);`,
		`CREATE INDEX IF NOT EXISTS idx_doc_nodes_type ON doc_nodes(type);`,
		`CREATE INDEX IF NOT EXISTS idx_doc_nodes_deleted ON doc_nodes(deleted_at);`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS docs_fts USING fts5(
			doc_id UNINDEXED,
			path,
			title,
			content,
			tokenize = 'unicode61'
		);`,
	}

	for _, q := range queries {
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("failed executing migration v3 query: %w\nQuery: %s", err, q)
		}
	}
	return nil
}

func migrationV4DocTags(tx *sql.Tx) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS doc_tags (
			doc_id     TEXT NOT NULL,
			tag_id     TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (doc_id, tag_id),
			FOREIGN KEY (doc_id) REFERENCES doc_nodes(id) ON DELETE CASCADE,
			FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_doc_tags_tag ON doc_tags(tag_id);`,
		`CREATE INDEX IF NOT EXISTS idx_doc_tags_doc ON doc_tags(doc_id);`,
	}

	for _, q := range queries {
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("failed executing migration v4 query: %w\nQuery: %s", err, q)
		}
	}
	return nil
}

func getTableColumns(tx *sql.Tx, tableName string) (map[string]bool, error) {
	rows, err := tx.Query(fmt.Sprintf("PRAGMA table_info(%s);", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		cols[name] = true
	}
	return cols, rows.Err()
}

// ApplyPluginMigration applies a versioned migration for a specific plugin.
func ApplyPluginMigration(pluginID string, version int, sqlScript string) error {
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return fmt.Errorf("pluginID cannot be empty")
	}

	hasher := sha256.New()
	hasher.Write([]byte(sqlScript))
	checksum := hex.EncodeToString(hasher.Sum(nil))

	var existingChecksum string
	err := DB.QueryRow(`SELECT checksum FROM plugin_migrations WHERE plugin_id = ? AND version = ?`, pluginID, version).Scan(&existingChecksum)
	if err == nil {
		if existingChecksum != checksum {
			return fmt.Errorf("migration version %d for plugin %q already applied with differing checksum %s (current: %s)", version, pluginID, existingChecksum, checksum)
		}
		return nil // already applied with identical checksum
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("failed checking plugin_migrations: %w", err)
	}

	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("failed starting transaction for plugin migration: %w", err)
	}

	if _, err := tx.Exec(sqlScript); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed executing plugin migration %q v%d: %w", pluginID, version, err)
	}

	now := time.Now()
	_, err = tx.Exec(`INSERT INTO plugin_migrations (plugin_id, version, checksum, applied_at) VALUES (?, ?, ?, ?)`, pluginID, version, checksum, now)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed recording plugin migration %q v%d: %w", pluginID, version, err)
	}

	return tx.Commit()
}

// GetAppliedMigrations returns all recorded core migrations.
func GetAppliedMigrations() ([]MigrationRecord, error) {
	rows, err := DB.Query(`SELECT version, name, applied_at FROM schema_migrations ORDER BY version ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []MigrationRecord
	for rows.Next() {
		var r MigrationRecord
		if err := rows.Scan(&r.Version, &r.Name, &r.AppliedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// GetPluginMigrations returns all migrations recorded for a plugin.
func GetPluginMigrations(pluginID string) ([]PluginMigrationRecord, error) {
	rows, err := DB.Query(`SELECT plugin_id, version, checksum, applied_at FROM plugin_migrations WHERE plugin_id = ? ORDER BY version ASC`, pluginID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []PluginMigrationRecord
	for rows.Next() {
		var r PluginMigrationRecord
		if err := rows.Scan(&r.PluginID, &r.Version, &r.Checksum, &r.AppliedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}
