package sync

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
)

type PostgresClient struct {
	DB      *sql.DB
	ConnStr string
}

type SyncStats struct {
	NotesPushed     int           `json:"notes_pushed"`
	NotesPulled     int           `json:"notes_pulled"`
	TagsPushed      int           `json:"tags_pushed"`
	TagsPulled      int           `json:"tags_pulled"`
	LinksPushed     int           `json:"links_pushed"`
	LinksPulled     int           `json:"links_pulled"`
	LogsPushed      int           `json:"logs_pushed"`
	LogsPulled      int           `json:"logs_pulled"`
	AuditLogsPushed int           `json:"audit_logs_pushed"`
	AuditLogsPulled int           `json:"audit_logs_pulled"`
	Duration        time.Duration `json:"duration"`
}

func (s *SyncStats) TotalPushed() int {
	return s.NotesPushed + s.TagsPushed + s.LinksPushed + s.LogsPushed + s.AuditLogsPushed
}

func (s *SyncStats) TotalPulled() int {
	return s.NotesPulled + s.TagsPulled + s.LinksPulled + s.LogsPulled + s.AuditLogsPulled
}

const pgInitSchemaSQL = `
CREATE TABLE IF NOT EXISTS notes (
  id VARCHAR(64) PRIMARY KEY,
  note TEXT NOT NULL,
  note_flesh TEXT DEFAULT '',
  type VARCHAR(64) DEFAULT 'note',
  status VARCHAR(64) DEFAULT 'active',
  area VARCHAR(64) DEFAULT '',
  importance INTEGER DEFAULT 0,
  clarity INTEGER DEFAULT 0,
  source TEXT DEFAULT '',
  target_date_time TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  deleted_note TEXT
);

CREATE TABLE IF NOT EXISTS tags (
  id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(255) UNIQUE NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS note_tags (
  note_id VARCHAR(64) REFERENCES notes(id) ON DELETE CASCADE,
  tag_id VARCHAR(64) REFERENCES tags(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (note_id, tag_id)
);

CREATE TABLE IF NOT EXISTS links (
  id VARCHAR(64) PRIMARY KEY,
  from_note VARCHAR(64) REFERENCES notes(id) ON DELETE CASCADE,
  to_note VARCHAR(64) REFERENCES notes(id) ON DELETE CASCADE,
  type VARCHAR(64) DEFAULT 'related_to',
  created_at TIMESTAMPTZ DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  deleted_note TEXT
);

CREATE TABLE IF NOT EXISTS daily_logs (
  id VARCHAR(64) PRIMARY KEY,
  content TEXT NOT NULL,
  note_id VARCHAR(64),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  deleted_note TEXT
);

CREATE TABLE IF NOT EXISTS audit_logs (
  id VARCHAR(64) PRIMARY KEY,
  entity_type VARCHAR(64) NOT NULL,
  entity_id VARCHAR(64) NOT NULL,
  action VARCHAR(64) NOT NULL,
  changes_summary TEXT,
  snapshot_json TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pg_notes_deleted_at ON notes(deleted_at);
CREATE INDEX IF NOT EXISTS idx_pg_notes_updated_at ON notes(updated_at);
CREATE INDEX IF NOT EXISTS idx_pg_notes_type ON notes(type);
CREATE INDEX IF NOT EXISTS idx_pg_notes_status ON notes(status);
CREATE INDEX IF NOT EXISTS idx_pg_daily_logs_created_at ON daily_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_pg_links_from_to ON links(from_note, to_note);
CREATE INDEX IF NOT EXISTS idx_pg_audit_entity ON audit_logs(entity_type, entity_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pg_audit_created_at ON audit_logs(created_at DESC);
`

// NewPostgresClient creates and connects to a remote PostgreSQL database.
func NewPostgresClient(connStr string) (*PostgresClient, error) {
	connStr = strings.TrimSpace(connStr)
	if connStr == "" {
		return nil, fmt.Errorf("postgres connection string is empty")
	}

	pgDB, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize postgres driver: %w", err)
	}

	pgDB.SetMaxOpenConns(5)
	pgDB.SetMaxIdleConns(2)
	pgDB.SetConnMaxLifetime(5 * time.Minute)

	return &PostgresClient{
		DB:      pgDB,
		ConnStr: connStr,
	}, nil
}

// Close closes the database connection.
func (c *PostgresClient) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}

// TestConnection verifies connectivity to PostgreSQL.
func (c *PostgresClient) TestConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.DB.PingContext(ctx); err != nil {
		return fmt.Errorf("postgres connection failed: %w", err)
	}
	return nil
}

// InitRemoteSchema creates any missing tables and indexes on the remote PostgreSQL instance.
func (c *PostgresClient) InitRemoteSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := c.DB.ExecContext(ctx, pgInitSchemaSQL)
	if err != nil {
		return fmt.Errorf("failed initializing remote postgres schema: %w", err)
	}
	return nil
}

// PushLocalChanges uploads modified notes, tags, links, and logs to PostgreSQL.
func (c *PostgresClient) PushLocalChanges(since time.Time) (*SyncStats, error) {
	stats := &SyncStats{}

	// 1. Push Notes
	localNotes, err := db.GetAllNotesSince(since)
	if err != nil {
		return stats, fmt.Errorf("failed reading local notes: %w", err)
	}
	for _, n := range localNotes {
		var targetDT sql.NullTime
		if !n.TargetDateTime.IsZero() {
			targetDT = sql.NullTime{Time: n.TargetDateTime, Valid: true}
		}
		var deletedDT sql.NullTime
		if !n.DeletedAt.IsZero() {
			deletedDT = sql.NullTime{Time: n.DeletedAt, Valid: true}
		}

		query := `
			INSERT INTO notes (
				id, note, note_flesh, type, status, area, importance, clarity, source, target_date_time, created_at, updated_at, deleted_at, deleted_note
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			ON CONFLICT (id) DO UPDATE SET
				note = EXCLUDED.note,
				note_flesh = EXCLUDED.note_flesh,
				type = EXCLUDED.type,
				status = EXCLUDED.status,
				area = EXCLUDED.area,
				importance = EXCLUDED.importance,
				clarity = EXCLUDED.clarity,
				source = EXCLUDED.source,
				target_date_time = EXCLUDED.target_date_time,
				created_at = EXCLUDED.created_at,
				updated_at = EXCLUDED.updated_at,
				deleted_at = EXCLUDED.deleted_at,
				deleted_note = EXCLUDED.deleted_note
			WHERE EXCLUDED.updated_at >= notes.updated_at OR notes.updated_at IS NULL
		`
		_, err := c.DB.Exec(query, n.ID, n.Note, n.NoteFlesh, string(n.Type), string(n.Status), string(n.Area), n.Importance, n.Clarity, n.Source, targetDT, n.CreatedAt, n.UpdatedAt, deletedDT, n.DeletedNote)
		if err != nil {
			return stats, fmt.Errorf("failed pushing note [%s]: %w", n.ID[:7], err)
		}
		stats.NotesPushed++
	}

	// 2. Push Tags
	localTags, err := db.GetAllTagsSince(since)
	if err != nil {
		return stats, fmt.Errorf("failed reading local tags: %w", err)
	}
	for _, t := range localTags {
		query := `
			INSERT INTO tags (id, name, created_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (name) DO UPDATE SET
				name = EXCLUDED.name
		`
		_, err := c.DB.Exec(query, t.ID, t.Name, t.CreatedAt)
		if err != nil {
			return stats, fmt.Errorf("failed pushing tag %q: %w", t.Name, err)
		}
		stats.TagsPushed++
	}

	// 3. Push Note Tags
	localNoteTags, err := db.GetAllNoteTagsSince(since)
	if err != nil {
		return stats, fmt.Errorf("failed reading local note_tags: %w", err)
	}
	for _, nt := range localNoteTags {
		query := `
			INSERT INTO note_tags (note_id, tag_id, created_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (note_id, tag_id) DO NOTHING
		`
		if _, err := c.DB.Exec(query, nt.NoteID, nt.TagID, nt.CreatedAt); err != nil {
			return stats, fmt.Errorf("failed pushing note_tag: %w", err)
		}
	}

	// 4. Push Links
	localLinks, err := db.GetAllLinksSince(since)
	if err != nil {
		return stats, fmt.Errorf("failed reading local links: %w", err)
	}
	for _, l := range localLinks {
		var deletedDT sql.NullTime
		if !l.DeletedAt.IsZero() {
			deletedDT = sql.NullTime{Time: l.DeletedAt, Valid: true}
		}

		query := `
			INSERT INTO links (id, from_note, to_note, type, created_at, deleted_at, deleted_note)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO UPDATE SET
				from_note = EXCLUDED.from_note,
				to_note = EXCLUDED.to_note,
				type = EXCLUDED.type,
				created_at = EXCLUDED.created_at,
				deleted_at = EXCLUDED.deleted_at,
				deleted_note = EXCLUDED.deleted_note
			WHERE (EXCLUDED.deleted_at IS NOT NULL AND (links.deleted_at IS NULL OR EXCLUDED.deleted_at >= links.deleted_at))
			   OR (EXCLUDED.created_at >= links.created_at OR links.created_at IS NULL)
		`
		_, err := c.DB.Exec(query, l.ID, l.FromNote, l.ToNote, string(l.Type), l.CreatedAt, deletedDT, l.DeletedNote)
		if err != nil {
			return stats, fmt.Errorf("failed pushing link [%s]: %w", l.ID[:7], err)
		}
		stats.LinksPushed++
	}

	// 5. Push Daily Logs
	localLogs, err := db.GetAllDailyLogsSince(since)
	if err != nil {
		return stats, fmt.Errorf("failed reading local daily logs: %w", err)
	}
	for _, l := range localLogs {
		var noteID sql.NullString
		if l.NoteID != "" {
			noteID = sql.NullString{String: l.NoteID, Valid: true}
		}
		var deletedDT sql.NullTime
		if !l.DeletedAt.IsZero() {
			deletedDT = sql.NullTime{Time: l.DeletedAt, Valid: true}
		}

		query := `
			INSERT INTO daily_logs (id, content, note_id, created_at, deleted_at, deleted_note)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO UPDATE SET
				content = EXCLUDED.content,
				note_id = EXCLUDED.note_id,
				created_at = EXCLUDED.created_at,
				deleted_at = EXCLUDED.deleted_at,
				deleted_note = EXCLUDED.deleted_note
			WHERE (EXCLUDED.deleted_at IS NOT NULL AND (daily_logs.deleted_at IS NULL OR EXCLUDED.deleted_at >= daily_logs.deleted_at))
			   OR (EXCLUDED.created_at >= daily_logs.created_at OR daily_logs.created_at IS NULL)
		`
		_, err := c.DB.Exec(query, l.ID, l.Content, noteID, l.CreatedAt, deletedDT, l.DeletedNote)
		if err != nil {
			return stats, fmt.Errorf("failed pushing daily log [%s]: %w", l.ID[:7], err)
		}
		stats.LogsPushed++
	}

	// 6. Push Audit Logs
	localAudits, err := db.GetAllAuditLogsSince(since)
	if err != nil {
		return stats, fmt.Errorf("failed reading local audit logs: %w", err)
	}
	for _, a := range localAudits {
		query := `
			INSERT INTO audit_logs (id, entity_type, entity_id, action, changes_summary, snapshot_json, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO NOTHING
		`
		_, err := c.DB.Exec(query, a.ID, a.EntityType, a.EntityID, string(a.Action), a.ChangesSummary, a.SnapshotJSON, a.CreatedAt)
		if err != nil {
			return stats, fmt.Errorf("failed pushing audit log [%s]: %w", a.ID[:7], err)
		}
		stats.AuditLogsPushed++
	}

	return stats, nil
}

// PullRemoteChanges fetches modified notes, tags, links, logs, and audit logs from PostgreSQL and applies them locally.
func (c *PostgresClient) PullRemoteChanges(since time.Time) (*SyncStats, error) {
	stats := &SyncStats{}

	// 1. Pull Notes
	var noteQuery string
	var noteArgs []interface{}
	if since.IsZero() {
		noteQuery = `SELECT id, note, note_flesh, type, status, area, importance, clarity, source, target_date_time, created_at, updated_at, deleted_at, deleted_note FROM notes`
	} else {
		noteQuery = `SELECT id, note, note_flesh, type, status, area, importance, clarity, source, target_date_time, created_at, updated_at, deleted_at, deleted_note FROM notes WHERE updated_at > $1 OR (deleted_at IS NOT NULL AND deleted_at > $1)`
		noteArgs = append(noteArgs, since)
	}

	rows, err := c.DB.Query(noteQuery, noteArgs...)
	if err != nil {
		return stats, fmt.Errorf("failed querying remote notes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var n models.Note
		var targetDT sql.NullTime
		var deletedDT sql.NullTime
		var deletedNote sql.NullString
		if err := rows.Scan(&n.ID, &n.Note, &n.NoteFlesh, &n.Type, &n.Status, &n.Area, &n.Importance, &n.Clarity, &n.Source, &targetDT, &n.CreatedAt, &n.UpdatedAt, &deletedDT, &deletedNote); err != nil {
			return stats, err
		}
		if targetDT.Valid {
			n.TargetDateTime = targetDT.Time
		}
		if deletedDT.Valid {
			n.DeletedAt = deletedDT.Time
		}
		if deletedNote.Valid {
			n.DeletedNote = deletedNote.String
		}
		if err := db.UpsertRemoteNote(&n); err != nil {
			return stats, fmt.Errorf("failed storing pulled note [%s]: %w", n.ID[:7], err)
		}
		stats.NotesPulled++
	}
	rows.Close()

	// 2. Pull Tags
	var tagQuery string
	var tagArgs []interface{}
	if since.IsZero() {
		tagQuery = `SELECT id, name, created_at FROM tags`
	} else {
		tagQuery = `SELECT id, name, created_at FROM tags WHERE created_at > $1`
		tagArgs = append(tagArgs, since)
	}

	tRows, err := c.DB.Query(tagQuery, tagArgs...)
	if err != nil {
		return stats, fmt.Errorf("failed querying remote tags: %w", err)
	}
	defer tRows.Close()

	for tRows.Next() {
		var t models.Tag
		var createdAt sql.NullTime
		if err := tRows.Scan(&t.ID, &t.Name, &createdAt); err != nil {
			return stats, err
		}
		if createdAt.Valid {
			t.CreatedAt = createdAt.Time
		}
		if err := db.UpsertRemoteTag(&t); err != nil {
			return stats, fmt.Errorf("failed storing pulled tag: %w", err)
		}
		stats.TagsPulled++
	}
	tRows.Close()

	// 3. Pull Note Tags
	var ntQuery string
	var ntArgs []interface{}
	if since.IsZero() {
		ntQuery = `SELECT note_id, tag_id, created_at FROM note_tags`
	} else {
		ntQuery = `SELECT note_id, tag_id, created_at FROM note_tags WHERE created_at > $1`
		ntArgs = append(ntArgs, since)
	}

	ntRows, err := c.DB.Query(ntQuery, ntArgs...)
	if err != nil {
		return stats, fmt.Errorf("failed querying remote note_tags: %w", err)
	}
	defer ntRows.Close()

	for ntRows.Next() {
		var nt models.NoteTag
		var createdAt sql.NullTime
		if err := ntRows.Scan(&nt.NoteID, &nt.TagID, &createdAt); err != nil {
			return stats, err
		}
		if createdAt.Valid {
			nt.CreatedAt = createdAt.Time
		}
		if err := db.UpsertRemoteNoteTag(&nt); err != nil {
			return stats, fmt.Errorf("failed storing pulled note_tag: %w", err)
		}
	}
	ntRows.Close()

	// 4. Pull Links
	var lQuery string
	var lArgs []interface{}
	if since.IsZero() {
		lQuery = `SELECT id, from_note, to_note, type, created_at, deleted_at, deleted_note FROM links`
	} else {
		lQuery = `SELECT id, from_note, to_note, type, created_at, deleted_at, deleted_note FROM links WHERE created_at > $1 OR (deleted_at IS NOT NULL AND deleted_at > $1)`
		lArgs = append(lArgs, since)
	}

	lRows, err := c.DB.Query(lQuery, lArgs...)
	if err != nil {
		return stats, fmt.Errorf("failed querying remote links: %w", err)
	}
	defer lRows.Close()

	for lRows.Next() {
		var l models.Link
		var createdAt sql.NullTime
		var deletedDT sql.NullTime
		var deletedNote sql.NullString
		if err := lRows.Scan(&l.ID, &l.FromNote, &l.ToNote, &l.Type, &createdAt, &deletedDT, &deletedNote); err != nil {
			return stats, err
		}
		if createdAt.Valid {
			l.CreatedAt = createdAt.Time
		}
		if deletedDT.Valid {
			l.DeletedAt = deletedDT.Time
		}
		if deletedNote.Valid {
			l.DeletedNote = deletedNote.String
		}
		if err := db.UpsertRemoteLink(&l); err != nil {
			return stats, fmt.Errorf("failed storing pulled link: %w", err)
		}
		stats.LinksPulled++
	}
	lRows.Close()

	// 5. Pull Daily Logs
	var logQuery string
	var logArgs []interface{}
	if since.IsZero() {
		logQuery = `SELECT id, content, note_id, created_at, deleted_at, deleted_note FROM daily_logs`
	} else {
		logQuery = `SELECT id, content, note_id, created_at, deleted_at, deleted_note FROM daily_logs WHERE created_at > $1 OR (deleted_at IS NOT NULL AND deleted_at > $1)`
		logArgs = append(logArgs, since)
	}

	logRows, err := c.DB.Query(logQuery, logArgs...)
	if err != nil {
		return stats, fmt.Errorf("failed querying remote daily logs: %w", err)
	}
	defer logRows.Close()

	for logRows.Next() {
		var l models.DailyLog
		var noteID sql.NullString
		var deletedDT sql.NullTime
		var deletedNote sql.NullString
		if err := logRows.Scan(&l.ID, &l.Content, &noteID, &l.CreatedAt, &deletedDT, &deletedNote); err != nil {
			return stats, err
		}
		if noteID.Valid {
			l.NoteID = noteID.String
		}
		if deletedDT.Valid {
			l.DeletedAt = deletedDT.Time
		}
		if deletedNote.Valid {
			l.DeletedNote = deletedNote.String
		}
		if err := db.UpsertRemoteDailyLog(&l); err != nil {
			return stats, fmt.Errorf("failed storing pulled daily log: %w", err)
		}
		stats.LogsPulled++
	}
	logRows.Close()

	// 6. Pull Audit Logs
	var auditQuery string
	var auditArgs []interface{}
	if since.IsZero() {
		auditQuery = `SELECT id, entity_type, entity_id, action, changes_summary, snapshot_json, created_at FROM audit_logs`
	} else {
		auditQuery = `SELECT id, entity_type, entity_id, action, changes_summary, snapshot_json, created_at FROM audit_logs WHERE created_at > $1`
		auditArgs = append(auditArgs, since)
	}

	aRows, err := c.DB.Query(auditQuery, auditArgs...)
	if err != nil {
		return stats, fmt.Errorf("failed querying remote audit logs: %w", err)
	}
	defer aRows.Close()

	for aRows.Next() {
		var a models.AuditEntry
		var summary sql.NullString
		var snapshot sql.NullString
		var action string
		if err := aRows.Scan(&a.ID, &a.EntityType, &a.EntityID, &action, &summary, &snapshot, &a.CreatedAt); err != nil {
			return stats, err
		}
		a.Action = models.AuditAction(action)
		if summary.Valid {
			a.ChangesSummary = summary.String
		}
		if snapshot.Valid {
			a.SnapshotJSON = snapshot.String
		}
		if err := db.UpsertRemoteAuditLog(&a); err != nil {
			return stats, fmt.Errorf("failed storing pulled audit log: %w", err)
		}
		stats.AuditLogsPulled++
	}
	aRows.Close()

	return stats, nil
}

// TwoWaySync performs a schema check, pushes local changes, pulls remote changes, and updates sync state.
func (c *PostgresClient) TwoWaySync() (*SyncStats, error) {
	start := time.Now()

	// 1. Ensure remote tables exist
	if err := c.InitRemoteSchema(); err != nil {
		return nil, err
	}

	// 2. Get last synced timestamp
	lastSync, err := db.GetLastSyncedAt("postgres")
	if err != nil {
		return nil, fmt.Errorf("failed reading sync state: %w", err)
	}

	// 3. Push local changes
	pushStats, err := c.PushLocalChanges(lastSync)
	if err != nil {
		return nil, err
	}

	// 4. Pull remote changes
	pullStats, err := c.PullRemoteChanges(lastSync)
	if err != nil {
		return nil, err
	}

	// 5. Update last sync time
	now := time.Now()
	if err := db.SetLastSyncedAt("postgres", now); err != nil {
		return nil, fmt.Errorf("failed updating sync state: %w", err)
	}

	combined := &SyncStats{
		NotesPushed:     pushStats.NotesPushed,
		NotesPulled:     pullStats.NotesPulled,
		TagsPushed:      pushStats.TagsPushed,
		TagsPulled:      pullStats.TagsPulled,
		LinksPushed:     pushStats.LinksPushed,
		LinksPulled:     pullStats.LinksPulled,
		LogsPushed:      pushStats.LogsPushed,
		LogsPulled:      pullStats.LogsPulled,
		AuditLogsPushed: pushStats.AuditLogsPushed,
		AuditLogsPulled: pullStats.AuditLogsPulled,
		Duration:        time.Since(start),
	}

	return combined, nil
}

