package db

import (
	"database/sql"
	"time"

	"github.com/tsusheel/kb-cli/models"
)

// GetAllNotesSince retrieves notes updated or deleted after the given time (or all if since is zero).
func GetAllNotesSince(since time.Time) ([]models.Note, error) {
	var query string
	var args []interface{}

	if since.IsZero() {
		query = `SELECT id, note, note_flesh, type, status, area, importance, clarity, source, target_date_time, created_at, updated_at, deleted_at, deleted_note FROM notes`
	} else {
		query = `SELECT id, note, note_flesh, type, status, area, importance, clarity, source, target_date_time, created_at, updated_at, deleted_at, deleted_note FROM notes WHERE updated_at > ? OR (deleted_at IS NOT NULL AND deleted_at > ?)`
		args = []interface{}{since, since}
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		var n models.Note
		var targetDT sql.NullTime
		var deletedDT sql.NullTime
		var deletedNote sql.NullString
		if err := rows.Scan(&n.ID, &n.Note, &n.NoteFlesh, &n.Type, &n.Status, &n.Area, &n.Importance, &n.Clarity, &n.Source, &targetDT, &n.CreatedAt, &n.UpdatedAt, &deletedDT, &deletedNote); err != nil {
			return nil, err
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
		notes = append(notes, n)
	}

	return notes, nil
}

// GetAllTagsSince retrieves tags created after the given time (or all if since is zero).
func GetAllTagsSince(since time.Time) ([]models.Tag, error) {
	var query string
	var args []interface{}

	if since.IsZero() {
		query = `SELECT id, name, created_at FROM tags`
	} else {
		query = `SELECT id, name, created_at FROM tags WHERE created_at > ?`
		args = []interface{}{since}
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		var createdAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.Name, &createdAt); err != nil {
			return nil, err
		}
		if createdAt.Valid {
			t.CreatedAt = createdAt.Time
		}
		tags = append(tags, t)
	}
	return tags, nil
}

// GetAllNoteTagsSince retrieves note_tags associations created after the given time (or all if since is zero).
func GetAllNoteTagsSince(since time.Time) ([]models.NoteTag, error) {
	var query string
	var args []interface{}

	if since.IsZero() {
		query = `SELECT note_id, tag_id, created_at FROM note_tags`
	} else {
		query = `SELECT note_id, tag_id, created_at FROM note_tags WHERE created_at > ?`
		args = []interface{}{since}
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var noteTags []models.NoteTag
	for rows.Next() {
		var nt models.NoteTag
		var createdAt sql.NullTime
		if err := rows.Scan(&nt.NoteID, &nt.TagID, &createdAt); err != nil {
			return nil, err
		}
		if createdAt.Valid {
			nt.CreatedAt = createdAt.Time
		}
		noteTags = append(noteTags, nt)
	}
	return noteTags, nil
}

// GetAllLinksSince retrieves links created or deleted after the given time (or all if since is zero).
func GetAllLinksSince(since time.Time) ([]models.Link, error) {
	var query string
	var args []interface{}

	if since.IsZero() {
		query = `SELECT id, from_note, to_note, type, created_at, deleted_at, deleted_note FROM links`
	} else {
		query = `SELECT id, from_note, to_note, type, created_at, deleted_at, deleted_note FROM links WHERE created_at > ? OR (deleted_at IS NOT NULL AND deleted_at > ?)`
		args = []interface{}{since, since}
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.Link
	for rows.Next() {
		var l models.Link
		var createdAt sql.NullTime
		var deletedAt sql.NullTime
		var deletedNote sql.NullString
		if err := rows.Scan(&l.ID, &l.FromNote, &l.ToNote, &l.Type, &createdAt, &deletedAt, &deletedNote); err != nil {
			return nil, err
		}
		if createdAt.Valid {
			l.CreatedAt = createdAt.Time
		}
		if deletedAt.Valid {
			l.DeletedAt = deletedAt.Time
		}
		if deletedNote.Valid {
			l.DeletedNote = deletedNote.String
		}
		links = append(links, l)
	}
	return links, nil
}

// GetAllDailyLogsSince retrieves daily logs created or deleted after the given time (or all if since is zero).
func GetAllDailyLogsSince(since time.Time) ([]models.DailyLog, error) {
	var query string
	var args []interface{}

	if since.IsZero() {
		query = `SELECT id, content, note_id, created_at, deleted_at, deleted_note FROM daily_logs`
	} else {
		query = `SELECT id, content, note_id, created_at, deleted_at, deleted_note FROM daily_logs WHERE created_at > ? OR (deleted_at IS NOT NULL AND deleted_at > ?)`
		args = []interface{}{since, since}
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.DailyLog
	for rows.Next() {
		var l models.DailyLog
		var noteID sql.NullString
		var deletedAt sql.NullTime
		var deletedNote sql.NullString
		if err := rows.Scan(&l.ID, &l.Content, &noteID, &l.CreatedAt, &deletedAt, &deletedNote); err != nil {
			return nil, err
		}
		if noteID.Valid {
			l.NoteID = noteID.String
		}
		if deletedAt.Valid {
			l.DeletedAt = deletedAt.Time
		}
		if deletedNote.Valid {
			l.DeletedNote = deletedNote.String
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// UpsertRemoteNote inserts or updates a note pulled from the remote database and updates FTS.
func UpsertRemoteNote(n *models.Note) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

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
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			note = excluded.note,
			note_flesh = excluded.note_flesh,
			type = excluded.type,
			status = excluded.status,
			area = excluded.area,
			importance = excluded.importance,
			clarity = excluded.clarity,
			source = excluded.source,
			target_date_time = excluded.target_date_time,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			deleted_at = excluded.deleted_at,
			deleted_note = excluded.deleted_note
	`
	_, err = tx.Exec(query, n.ID, n.Note, n.NoteFlesh, n.Type, n.Status, n.Area, n.Importance, n.Clarity, n.Source, targetDT, n.CreatedAt, n.UpdatedAt, deletedDT, n.DeletedNote)
	if err != nil {
		return err
	}

	// Refresh FTS
	tx.Exec(`DELETE FROM notes_fts WHERE note_id = ?`, n.ID)
	if n.DeletedAt.IsZero() {
		tx.Exec(`INSERT INTO notes_fts (note_id, note, note_flesh) VALUES (?, ?, ?)`, n.ID, n.Note, n.NoteFlesh)
	}

	return tx.Commit()
}

// UpsertRemoteTag inserts a tag if it doesn't already exist.
func UpsertRemoteTag(t *models.Tag) error {
	query := `
		INSERT INTO tags (id, name, created_at)
		VALUES (?, ?, ?)
		ON CONFLICT(name) DO NOTHING
	`
	_, err := DB.Exec(query, t.ID, t.Name, t.CreatedAt)
	return err
}

// UpsertRemoteNoteTag inserts a note-tag junction row if not present.
func UpsertRemoteNoteTag(nt *models.NoteTag) error {
	query := `
		INSERT OR IGNORE INTO note_tags (note_id, tag_id, created_at)
		VALUES (?, ?, ?)
	`
	_, err := DB.Exec(query, nt.NoteID, nt.TagID, nt.CreatedAt)
	return err
}

// UpsertRemoteLink inserts or updates a link pulled from the remote database.
func UpsertRemoteLink(l *models.Link) error {
	var deletedDT sql.NullTime
	if !l.DeletedAt.IsZero() {
		deletedDT = sql.NullTime{Time: l.DeletedAt, Valid: true}
	}

	query := `
		INSERT INTO links (id, from_note, to_note, type, created_at, deleted_at, deleted_note)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			from_note = excluded.from_note,
			to_note = excluded.to_note,
			type = excluded.type,
			created_at = excluded.created_at,
			deleted_at = excluded.deleted_at,
			deleted_note = excluded.deleted_note
	`
	_, err := DB.Exec(query, l.ID, l.FromNote, l.ToNote, l.Type, l.CreatedAt, deletedDT, l.DeletedNote)
	return err
}

// UpsertRemoteDailyLog inserts or updates a daily log pulled from remote.
func UpsertRemoteDailyLog(l *models.DailyLog) error {
	var deletedDT sql.NullTime
	if !l.DeletedAt.IsZero() {
		deletedDT = sql.NullTime{Time: l.DeletedAt, Valid: true}
	}

	query := `
		INSERT INTO daily_logs (id, content, note_id, created_at, deleted_at, deleted_note)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			content = excluded.content,
			note_id = excluded.note_id,
			created_at = excluded.created_at,
			deleted_at = excluded.deleted_at,
			deleted_note = excluded.deleted_note
	`
	_, err := DB.Exec(query, l.ID, l.Content, l.NoteID, l.CreatedAt, deletedDT, l.DeletedNote)
	return err
}

// GetAllAuditLogsSince retrieves audit logs created after the given time (or all if since is zero).
func GetAllAuditLogsSince(since time.Time) ([]models.AuditEntry, error) {
	var query string
	var args []interface{}

	if since.IsZero() {
		query = `SELECT id, entity_type, entity_id, action, changes_summary, snapshot_json, created_at FROM audit_logs`
	} else {
		query = `SELECT id, entity_type, entity_id, action, changes_summary, snapshot_json, created_at FROM audit_logs WHERE created_at > ?`
		args = []interface{}{since}
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.AuditEntry
	for rows.Next() {
		var entry models.AuditEntry
		var summary sql.NullString
		var snapshot sql.NullString
		var action string

		if err := rows.Scan(&entry.ID, &entry.EntityType, &entry.EntityID, &action, &summary, &snapshot, &entry.CreatedAt); err != nil {
			return nil, err
		}
		entry.Action = models.AuditAction(action)
		if summary.Valid {
			entry.ChangesSummary = summary.String
		}
		if snapshot.Valid {
			entry.SnapshotJSON = snapshot.String
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// UpsertRemoteAuditLog inserts an audit log entry pulled from remote if not already present.
func UpsertRemoteAuditLog(entry *models.AuditEntry) error {
	query := `
		INSERT INTO audit_logs (id, entity_type, entity_id, action, changes_summary, snapshot_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`
	_, err := DB.Exec(query, entry.ID, entry.EntityType, entry.EntityID, string(entry.Action), entry.ChangesSummary, entry.SnapshotJSON, entry.CreatedAt)
	return err
}

