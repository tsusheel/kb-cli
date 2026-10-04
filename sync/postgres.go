package sync

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
	stdsync "sync"
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

// EnsureRemoteSchema checks if the schema is already initialized before running DDL.
func (c *PostgresClient) EnsureRemoteSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var dummy int
	err := c.DB.QueryRowContext(ctx, "SELECT 1 FROM notes LIMIT 1").Scan(&dummy)
	if err == nil || err == sql.ErrNoRows {
		return nil
	}
	return c.InitRemoteSchema()
}

func batchPushNotes(tx *sql.Tx, notes []models.Note) error {
	const batchSize = 50
	for i := 0; i < len(notes); i += batchSize {
		end := i + batchSize
		if end > len(notes) {
			end = len(notes)
		}
		chunk := notes[i:end]

		var valueStrings []string
		var valueArgs []interface{}
		colCount := 14

		for j, n := range chunk {
			var targetDT sql.NullTime
			if !n.TargetDateTime.IsZero() {
				targetDT = sql.NullTime{Time: n.TargetDateTime, Valid: true}
			}
			var deletedDT sql.NullTime
			if !n.DeletedAt.IsZero() {
				deletedDT = sql.NullTime{Time: n.DeletedAt, Valid: true}
			}

			startArg := j*colCount + 1
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				startArg, startArg+1, startArg+2, startArg+3, startArg+4, startArg+5, startArg+6, startArg+7, startArg+8, startArg+9, startArg+10, startArg+11, startArg+12, startArg+13))

			valueArgs = append(valueArgs, n.ID, n.Note, n.NoteFlesh, string(n.Type), string(n.Status), string(n.Area), n.Importance, n.Clarity, n.Source, targetDT, n.CreatedAt, n.UpdatedAt, deletedDT, n.DeletedNote)
		}

		query := fmt.Sprintf(`
			INSERT INTO notes (
				id, note, note_flesh, type, status, area, importance, clarity, source, target_date_time, created_at, updated_at, deleted_at, deleted_note
			) VALUES %s
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
		`, strings.Join(valueStrings, ","))

		if _, err := tx.Exec(query, valueArgs...); err != nil {
			return err
		}
	}
	return nil
}

func batchPushTags(tx *sql.Tx, tags []models.Tag) error {
	const batchSize = 100
	for i := 0; i < len(tags); i += batchSize {
		end := i + batchSize
		if end > len(tags) {
			end = len(tags)
		}
		chunk := tags[i:end]

		var valueStrings []string
		var valueArgs []interface{}
		colCount := 3

		for j, t := range chunk {
			startArg := j*colCount + 1
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d)", startArg, startArg+1, startArg+2))
			valueArgs = append(valueArgs, t.ID, t.Name, t.CreatedAt)
		}

		query := fmt.Sprintf(`
			INSERT INTO tags (id, name, created_at)
			VALUES %s
			ON CONFLICT (name) DO NOTHING
		`, strings.Join(valueStrings, ","))

		if _, err := tx.Exec(query, valueArgs...); err != nil {
			return err
		}
	}
	return nil
}

func batchDeleteNoteTags(tx *sql.Tx, noteIDs []string) error {
	const batchSize = 100
	for i := 0; i < len(noteIDs); i += batchSize {
		end := i + batchSize
		if end > len(noteIDs) {
			end = len(noteIDs)
		}
		chunk := noteIDs[i:end]

		placeholders := make([]string, len(chunk))
		args := make([]interface{}, len(chunk))
		for j, id := range chunk {
			placeholders[j] = fmt.Sprintf("$%d", j+1)
			args[j] = id
		}

		query := fmt.Sprintf("DELETE FROM note_tags WHERE note_id IN (%s)", strings.Join(placeholders, ","))
		if _, err := tx.Exec(query, args...); err != nil {
			return err
		}
	}
	return nil
}

func batchPushNoteTags(tx *sql.Tx, noteTags []models.NoteTag) error {
	const batchSize = 100
	for i := 0; i < len(noteTags); i += batchSize {
		end := i + batchSize
		if end > len(noteTags) {
			end = len(noteTags)
		}
		chunk := noteTags[i:end]

		var valueStrings []string
		var valueArgs []interface{}
		colCount := 3

		for j, nt := range chunk {
			startArg := j*colCount + 1
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d)", startArg, startArg+1, startArg+2))
			valueArgs = append(valueArgs, nt.NoteID, nt.TagID, nt.CreatedAt)
		}

		query := fmt.Sprintf(`
			INSERT INTO note_tags (note_id, tag_id, created_at)
			VALUES %s
			ON CONFLICT (note_id, tag_id) DO NOTHING
		`, strings.Join(valueStrings, ","))

		if _, err := tx.Exec(query, valueArgs...); err != nil {
			return err
		}
	}
	return nil
}

func batchPushLinks(tx *sql.Tx, links []models.Link) error {
	const batchSize = 50
	for i := 0; i < len(links); i += batchSize {
		end := i + batchSize
		if end > len(links) {
			end = len(links)
		}
		chunk := links[i:end]

		var valueStrings []string
		var valueArgs []interface{}
		colCount := 7

		for j, l := range chunk {
			var deletedDT sql.NullTime
			if !l.DeletedAt.IsZero() {
				deletedDT = sql.NullTime{Time: l.DeletedAt, Valid: true}
			}

			startArg := j*colCount + 1
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				startArg, startArg+1, startArg+2, startArg+3, startArg+4, startArg+5, startArg+6))
			valueArgs = append(valueArgs, l.ID, l.FromNote, l.ToNote, string(l.Type), l.CreatedAt, deletedDT, l.DeletedNote)
		}

		query := fmt.Sprintf(`
			INSERT INTO links (id, from_note, to_note, type, created_at, deleted_at, deleted_note)
			VALUES %s
			ON CONFLICT (id) DO UPDATE SET
				from_note = EXCLUDED.from_note,
				to_note = EXCLUDED.to_note,
				type = EXCLUDED.type,
				created_at = EXCLUDED.created_at,
				deleted_at = EXCLUDED.deleted_at,
				deleted_note = EXCLUDED.deleted_note
			WHERE (EXCLUDED.deleted_at IS NOT NULL AND (links.deleted_at IS NULL OR EXCLUDED.deleted_at >= links.deleted_at))
			   OR (links.deleted_at IS NULL)
		`, strings.Join(valueStrings, ","))

		if _, err := tx.Exec(query, valueArgs...); err != nil {
			return err
		}
	}
	return nil
}

func batchPushDailyLogs(tx *sql.Tx, logs []models.DailyLog) error {
	const batchSize = 50
	for i := 0; i < len(logs); i += batchSize {
		end := i + batchSize
		if end > len(logs) {
			end = len(logs)
		}
		chunk := logs[i:end]

		var valueStrings []string
		var valueArgs []interface{}
		colCount := 6

		for j, l := range chunk {
			var noteID sql.NullString
			if l.NoteID != "" {
				noteID = sql.NullString{String: l.NoteID, Valid: true}
			}
			var deletedDT sql.NullTime
			if !l.DeletedAt.IsZero() {
				deletedDT = sql.NullTime{Time: l.DeletedAt, Valid: true}
			}

			startArg := j*colCount + 1
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)",
				startArg, startArg+1, startArg+2, startArg+3, startArg+4, startArg+5))
			valueArgs = append(valueArgs, l.ID, l.Content, noteID, l.CreatedAt, deletedDT, l.DeletedNote)
		}

		query := fmt.Sprintf(`
			INSERT INTO daily_logs (id, content, note_id, created_at, deleted_at, deleted_note)
			VALUES %s
			ON CONFLICT (id) DO UPDATE SET
				content = EXCLUDED.content,
				note_id = EXCLUDED.note_id,
				created_at = EXCLUDED.created_at,
				deleted_at = EXCLUDED.deleted_at,
				deleted_note = EXCLUDED.deleted_note
			WHERE (EXCLUDED.deleted_at IS NOT NULL AND (daily_logs.deleted_at IS NULL OR EXCLUDED.deleted_at >= daily_logs.deleted_at))
			   OR (EXCLUDED.created_at >= daily_logs.created_at OR daily_logs.created_at IS NULL)
		`, strings.Join(valueStrings, ","))

		if _, err := tx.Exec(query, valueArgs...); err != nil {
			return err
		}
	}
	return nil
}

func batchPushAuditLogs(tx *sql.Tx, audits []models.AuditEntry) error {
	const batchSize = 50
	for i := 0; i < len(audits); i += batchSize {
		end := i + batchSize
		if end > len(audits) {
			end = len(audits)
		}
		chunk := audits[i:end]

		var valueStrings []string
		var valueArgs []interface{}
		colCount := 7

		for j, a := range chunk {
			startArg := j*colCount + 1
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				startArg, startArg+1, startArg+2, startArg+3, startArg+4, startArg+5, startArg+6))
			valueArgs = append(valueArgs, a.ID, a.EntityType, a.EntityID, string(a.Action), a.ChangesSummary, a.SnapshotJSON, a.CreatedAt)
		}

		query := fmt.Sprintf(`
			INSERT INTO audit_logs (id, entity_type, entity_id, action, changes_summary, snapshot_json, created_at)
			VALUES %s
			ON CONFLICT (id) DO NOTHING
		`, strings.Join(valueStrings, ","))

		if _, err := tx.Exec(query, valueArgs...); err != nil {
			return err
		}
	}
	return nil
}

// PushLocalChanges uploads modified notes, tags, links, and logs to PostgreSQL in a single batched transaction.
func (c *PostgresClient) PushLocalChanges(since time.Time) (*SyncStats, error) {
	stats := &SyncStats{}

	// 1. Fetch Local Modified Records
	localNotes, err := db.GetAllNotesSince(since)
	if err != nil {
		return stats, fmt.Errorf("failed reading local notes: %w", err)
	}

	localTags, err := db.GetAllTagsSince(since)
	if err != nil {
		return stats, fmt.Errorf("failed reading local tags: %w", err)
	}

	var modifiedNoteIDs []string
	for _, n := range localNotes {
		modifiedNoteIDs = append(modifiedNoteIDs, n.ID)
	}

	var localNoteTags []models.NoteTag
	if since.IsZero() {
		localNoteTags, err = db.GetAllNoteTagsSince(time.Time{})
	} else if len(modifiedNoteIDs) > 0 {
		localNoteTags, err = db.GetNoteTagsForNotes(modifiedNoteIDs)
	}
	if err != nil {
		return stats, fmt.Errorf("failed reading local note_tags: %w", err)
	}

	localLinks, err := db.GetAllLinksSince(since)
	if err != nil {
		return stats, fmt.Errorf("failed reading local links: %w", err)
	}

	localLogs, err := db.GetAllDailyLogsSince(since)
	if err != nil {
		return stats, fmt.Errorf("failed reading local daily logs: %w", err)
	}

	localAudits, err := db.GetAllAuditLogsSince(since)
	if err != nil {
		return stats, fmt.Errorf("failed reading local audit logs: %w", err)
	}

	// 2. Open remote transaction for fast atomic push
	tx, err := c.DB.Begin()
	if err != nil {
		return stats, fmt.Errorf("failed starting remote transaction: %w", err)
	}
	defer tx.Rollback()

	// Push Notes
	if len(localNotes) > 0 {
		if err := batchPushNotes(tx, localNotes); err != nil {
			return stats, fmt.Errorf("failed batch pushing notes: %w", err)
		}
		stats.NotesPushed = len(localNotes)
	}

	// Push Tags
	if len(localTags) > 0 {
		if err := batchPushTags(tx, localTags); err != nil {
			return stats, fmt.Errorf("failed batch pushing tags: %w", err)
		}
		stats.TagsPushed = len(localTags)
	}

	// Clean and Push Note Tags
	if len(modifiedNoteIDs) > 0 {
		if err := batchDeleteNoteTags(tx, modifiedNoteIDs); err != nil {
			return stats, fmt.Errorf("failed cleaning remote note_tags: %w", err)
		}
	}
	if len(localNoteTags) > 0 {
		if err := batchPushNoteTags(tx, localNoteTags); err != nil {
			return stats, fmt.Errorf("failed batch pushing note_tags: %w", err)
		}
	}

	// Push Links
	if len(localLinks) > 0 {
		if err := batchPushLinks(tx, localLinks); err != nil {
			return stats, fmt.Errorf("failed batch pushing links: %w", err)
		}
		stats.LinksPushed = len(localLinks)
	}

	// Push Daily Logs
	if len(localLogs) > 0 {
		if err := batchPushDailyLogs(tx, localLogs); err != nil {
			return stats, fmt.Errorf("failed batch pushing daily logs: %w", err)
		}
		stats.LogsPushed = len(localLogs)
	}

	// Push Audit Logs
	if len(localAudits) > 0 {
		if err := batchPushAuditLogs(tx, localAudits); err != nil {
			return stats, fmt.Errorf("failed batch pushing audit logs: %w", err)
		}
		stats.AuditLogsPushed = len(localAudits)
	}

	if err := tx.Commit(); err != nil {
		return stats, fmt.Errorf("failed committing remote transaction: %w", err)
	}

	return stats, nil
}

// PullRemoteChanges fetches modified notes, tags, links, logs, and audit logs from PostgreSQL and applies them locally.
func (c *PostgresClient) PullRemoteChanges(since time.Time) (*SyncStats, error) {
	stats := &SyncStats{}
	pullData := &db.RemotePullData{}

	var g stdsync.WaitGroup
	var errNote, errTag, errLink, errLog, errAudit error

	// 1. Concurrent Pull Notes
	g.Add(1)
	go func() {
		defer g.Done()
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
			errNote = fmt.Errorf("failed querying remote notes: %w", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var n models.Note
			var targetDT sql.NullTime
			var deletedDT sql.NullTime
			var deletedNote sql.NullString
			if err := rows.Scan(&n.ID, &n.Note, &n.NoteFlesh, &n.Type, &n.Status, &n.Area, &n.Importance, &n.Clarity, &n.Source, &targetDT, &n.CreatedAt, &n.UpdatedAt, &deletedDT, &deletedNote); err != nil {
				errNote = err
				return
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
			pullData.Notes = append(pullData.Notes, n)
			pullData.ClearedNotes = append(pullData.ClearedNotes, n.ID)
		}
	}()

	// 2. Concurrent Pull Tags
	g.Add(1)
	go func() {
		defer g.Done()
		var tagQuery string
		var tagArgs []interface{}
		if since.IsZero() {
			tagQuery = `SELECT id, name, created_at FROM tags`
		} else {
			tagQuery = `SELECT id, name, created_at FROM tags WHERE created_at > $1`
			tagArgs = append(tagArgs, since)
		}

		rows, err := c.DB.Query(tagQuery, tagArgs...)
		if err != nil {
			errTag = fmt.Errorf("failed querying remote tags: %w", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var t models.Tag
			var createdAt sql.NullTime
			if err := rows.Scan(&t.ID, &t.Name, &createdAt); err != nil {
				errTag = err
				return
			}
			if createdAt.Valid {
				t.CreatedAt = createdAt.Time
			}
			pullData.Tags = append(pullData.Tags, t)
		}
	}()

	// 3. Concurrent Pull Links
	g.Add(1)
	go func() {
		defer g.Done()
		var lQuery string
		var lArgs []interface{}
		if since.IsZero() {
			lQuery = `SELECT id, from_note, to_note, type, created_at, deleted_at, deleted_note FROM links`
		} else {
			lQuery = `SELECT id, from_note, to_note, type, created_at, deleted_at, deleted_note FROM links WHERE created_at > $1 OR (deleted_at IS NOT NULL AND deleted_at > $1)`
			lArgs = append(lArgs, since)
		}

		rows, err := c.DB.Query(lQuery, lArgs...)
		if err != nil {
			errLink = fmt.Errorf("failed querying remote links: %w", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var l models.Link
			var createdAt sql.NullTime
			var deletedDT sql.NullTime
			var deletedNote sql.NullString
			if err := rows.Scan(&l.ID, &l.FromNote, &l.ToNote, &l.Type, &createdAt, &deletedDT, &deletedNote); err != nil {
				errLink = err
				return
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
			pullData.Links = append(pullData.Links, l)
		}
	}()

	// 4. Concurrent Pull Daily Logs
	g.Add(1)
	go func() {
		defer g.Done()
		var logQuery string
		var logArgs []interface{}
		if since.IsZero() {
			logQuery = `SELECT id, content, note_id, created_at, deleted_at, deleted_note FROM daily_logs`
		} else {
			logQuery = `SELECT id, content, note_id, created_at, deleted_at, deleted_note FROM daily_logs WHERE created_at > $1 OR (deleted_at IS NOT NULL AND deleted_at > $1)`
			logArgs = append(logArgs, since)
		}

		rows, err := c.DB.Query(logQuery, logArgs...)
		if err != nil {
			errLog = fmt.Errorf("failed querying remote daily logs: %w", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var l models.DailyLog
			var noteID sql.NullString
			var deletedDT sql.NullTime
			var deletedNote sql.NullString
			if err := rows.Scan(&l.ID, &l.Content, &noteID, &l.CreatedAt, &deletedDT, &deletedNote); err != nil {
				errLog = err
				return
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
			pullData.DailyLogs = append(pullData.DailyLogs, l)
		}
	}()

	// 5. Concurrent Pull Audit Logs
	g.Add(1)
	go func() {
		defer g.Done()
		var auditQuery string
		var auditArgs []interface{}
		if since.IsZero() {
			auditQuery = `SELECT id, entity_type, entity_id, action, changes_summary, snapshot_json, created_at FROM audit_logs`
		} else {
			auditQuery = `SELECT id, entity_type, entity_id, action, changes_summary, snapshot_json, created_at FROM audit_logs WHERE created_at > $1`
			auditArgs = append(auditArgs, since)
		}

		rows, err := c.DB.Query(auditQuery, auditArgs...)
		if err != nil {
			errAudit = fmt.Errorf("failed querying remote audit logs: %w", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var a models.AuditEntry
			var summary sql.NullString
			var snapshot sql.NullString
			var action string
			if err := rows.Scan(&a.ID, &a.EntityType, &a.EntityID, &action, &summary, &snapshot, &a.CreatedAt); err != nil {
				errAudit = err
				return
			}
			a.Action = models.AuditAction(action)
			if summary.Valid {
				a.ChangesSummary = summary.String
			}
			if snapshot.Valid {
				a.SnapshotJSON = snapshot.String
			}
			pullData.AuditLogs = append(pullData.AuditLogs, a)
		}
	}()

	g.Wait()

	if errNote != nil {
		return stats, errNote
	}
	if errTag != nil {
		return stats, errTag
	}
	if errLink != nil {
		return stats, errLink
	}
	if errLog != nil {
		return stats, errLog
	}
	if errAudit != nil {
		return stats, errAudit
	}

	// 6. Pull Note Tags (for modified pulled notes or on initial sync)
	if since.IsZero() {
		ntRows, err := c.DB.Query(`SELECT note_id, tag_id, created_at FROM note_tags`)
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
			pullData.NoteTags = append(pullData.NoteTags, nt)
		}
	} else if len(pullData.ClearedNotes) > 0 {
		const batchSize = 100
		for i := 0; i < len(pullData.ClearedNotes); i += batchSize {
			end := i + batchSize
			if end > len(pullData.ClearedNotes) {
				end = len(pullData.ClearedNotes)
			}
			chunk := pullData.ClearedNotes[i:end]

			placeholders := make([]string, len(chunk))
			args := make([]interface{}, len(chunk))
			for j, id := range chunk {
				placeholders[j] = fmt.Sprintf("$%d", j+1)
				args[j] = id
			}

			ntQuery := fmt.Sprintf("SELECT note_id, tag_id, created_at FROM note_tags WHERE note_id IN (%s)", strings.Join(placeholders, ","))
			ntRows, err := c.DB.Query(ntQuery, args...)
			if err != nil {
				return stats, fmt.Errorf("failed querying remote note_tags chunk: %w", err)
			}

			for ntRows.Next() {
				var nt models.NoteTag
				var createdAt sql.NullTime
				if err := ntRows.Scan(&nt.NoteID, &nt.TagID, &createdAt); err != nil {
					ntRows.Close()
					return stats, err
				}
				if createdAt.Valid {
					nt.CreatedAt = createdAt.Time
				}
				pullData.NoteTags = append(pullData.NoteTags, nt)
			}
			ntRows.Close()
		}
	}

	// Apply all pulled records in single atomic local SQLite transaction
	if err := db.ApplyRemotePulls(pullData); err != nil {
		return stats, fmt.Errorf("failed applying remote changes locally: %w", err)
	}

	stats.NotesPulled = len(pullData.Notes)
	stats.TagsPulled = len(pullData.Tags)
	stats.LinksPulled = len(pullData.Links)
	stats.LogsPulled = len(pullData.DailyLogs)
	stats.AuditLogsPulled = len(pullData.AuditLogs)

	return stats, nil
}

// TwoWaySync performs a schema check, pulls remote changes first, pushes local changes, and updates sync state.
func (c *PostgresClient) TwoWaySync() (*SyncStats, error) {
	start := time.Now()

	// 1. Ensure remote tables exist (lightweight check)
	if err := c.EnsureRemoteSchema(); err != nil {
		return nil, err
	}

	// 2. Get last synced timestamp
	lastSync, err := db.GetLastSyncedAt("postgres")
	if err != nil {
		return nil, fmt.Errorf("failed reading sync state: %w", err)
	}

	// 3. Pull remote changes first to harmonize tag IDs and receive remote updates
	pullStats, err := c.PullRemoteChanges(lastSync)
	if err != nil {
		return nil, err
	}

	// 4. Push local changes
	pushStats, err := c.PushLocalChanges(lastSync)
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
