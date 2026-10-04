package db

import (
	"database/sql"
	"strings"
	"time"

	"github.com/tsusheel/kb-cli/models"
)

// GetAllNotesSince retrieves notes updated or deleted after the given time (or all if since is zero).
func GetAllNotesSince(since time.Time) ([]models.Note, error) {
	var query string
	var args []interface{}

	if since.IsZero() {
		query = `SELECT ` + noteColumns + ` FROM notes`
	} else {
		query = `SELECT ` + noteColumns + ` FROM notes WHERE updated_at > ? OR (deleted_at IS NOT NULL AND deleted_at > ?)`
		args = []interface{}{since, since}
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, *n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
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
	if err := rows.Err(); err != nil {
		return nil, err
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return noteTags, nil
}

// GetNoteTagsForNotes retrieves note_tags for the specified list of note IDs.
func GetNoteTagsForNotes(noteIDs []string) ([]models.NoteTag, error) {
	if len(noteIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(noteIDs))
	args := make([]interface{}, len(noteIDs))
	for i, id := range noteIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `SELECT note_id, tag_id, created_at FROM note_tags WHERE note_id IN (` + strings.Join(placeholders, ",") + `)`
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
	if err := rows.Err(); err != nil {
		return nil, err
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
		l, err := scanLink(rows)
		if err != nil {
			return nil, err
		}
		links = append(links, *l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return links, nil
}

// GetAllDailyLogsSince retrieves daily logs created or deleted after the given time (or all if since is zero).
func GetAllDailyLogsSince(since time.Time) ([]models.DailyLog, error) {
	var query string
	var args []interface{}

	if since.IsZero() {
		query = `SELECT ` + logColumns + ` FROM daily_logs`
	} else {
		query = `SELECT ` + logColumns + ` FROM daily_logs WHERE created_at > ? OR (deleted_at IS NOT NULL AND deleted_at > ?)`
		args = []interface{}{since, since}
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.DailyLog
	for rows.Next() {
		l, err := scanLog(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, *l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
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

// UpsertRemoteTag inserts a tag if it doesn't already exist or harmonizes tag IDs.
func UpsertRemoteTag(t *models.Tag) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var existingID string
	err = tx.QueryRow("SELECT id FROM tags WHERE name = ?", t.Name).Scan(&existingID)
	if err == sql.ErrNoRows {
		_, err = tx.Exec("INSERT INTO tags (id, name, created_at) VALUES (?, ?, ?)", t.ID, t.Name, t.CreatedAt)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if existingID != t.ID {
		// Harmonize existing local note_tags to reference the remote tag ID
		_, err = tx.Exec("UPDATE note_tags SET tag_id = ? WHERE tag_id = ?", t.ID, existingID)
		if err != nil {
			return err
		}
		// Update tag ID in tags table
		_, err = tx.Exec("UPDATE tags SET id = ? WHERE name = ?", t.ID, t.Name)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
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
		WHERE (excluded.deleted_at IS NOT NULL AND (links.deleted_at IS NULL OR excluded.deleted_at >= links.deleted_at))
		   OR (links.deleted_at IS NULL)
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
	if err := rows.Err(); err != nil {
		return nil, err
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

// RemotePullData encapsulates all pulled entities from remote database.
type RemotePullData struct {
	Notes        []models.Note
	Tags         []models.Tag
	NoteTags     []models.NoteTag
	Links        []models.Link
	DailyLogs    []models.DailyLog
	AuditLogs    []models.AuditEntry
	ClearedNotes []string
}

// ApplyRemotePulls applies all pulled entities atomically within a single local SQLite transaction.
func ApplyRemotePulls(data *RemotePullData) error {
	if data == nil {
		return nil
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Process Tags (harmonization & inserts)
	if len(data.Tags) > 0 {
		stmtTagCheck, err := tx.Prepare("SELECT id FROM tags WHERE name = ?")
		if err != nil {
			return err
		}
		defer stmtTagCheck.Close()

		stmtTagIns, err := tx.Prepare("INSERT INTO tags (id, name, created_at) VALUES (?, ?, ?)")
		if err != nil {
			return err
		}
		defer stmtTagIns.Close()

		stmtTagHarmonizeNT, err := tx.Prepare("UPDATE note_tags SET tag_id = ? WHERE tag_id = ?")
		if err != nil {
			return err
		}
		defer stmtTagHarmonizeNT.Close()

		stmtTagHarmonizeT, err := tx.Prepare("UPDATE tags SET id = ? WHERE name = ?")
		if err != nil {
			return err
		}
		defer stmtTagHarmonizeT.Close()

		for _, t := range data.Tags {
			var existingID string
			err = stmtTagCheck.QueryRow(t.Name).Scan(&existingID)
			if err == sql.ErrNoRows {
				if _, err := stmtTagIns.Exec(t.ID, t.Name, t.CreatedAt); err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else if existingID != t.ID {
				if _, err := stmtTagHarmonizeNT.Exec(t.ID, existingID); err != nil {
					return err
				}
				if _, err := stmtTagHarmonizeT.Exec(t.ID, t.Name); err != nil {
					return err
				}
			}
		}
	}

	// 2. Process Notes & FTS
	if len(data.Notes) > 0 {
		noteQuery := `
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
		stmtNote, err := tx.Prepare(noteQuery)
		if err != nil {
			return err
		}
		defer stmtNote.Close()

		stmtFtsDel, err := tx.Prepare("DELETE FROM notes_fts WHERE note_id = ?")
		if err != nil {
			return err
		}
		defer stmtFtsDel.Close()

		stmtFtsIns, err := tx.Prepare("INSERT INTO notes_fts (note_id, note, note_flesh) VALUES (?, ?, ?)")
		if err != nil {
			return err
		}
		defer stmtFtsIns.Close()

		for _, n := range data.Notes {
			cleanNoteID := strings.ReplaceAll(n.ID, "-", "")
			var targetDT sql.NullTime
			if !n.TargetDateTime.IsZero() {
				targetDT = sql.NullTime{Time: n.TargetDateTime, Valid: true}
			}
			var deletedDT sql.NullTime
			if !n.DeletedAt.IsZero() {
				deletedDT = sql.NullTime{Time: n.DeletedAt, Valid: true}
			}

			if _, err := stmtNote.Exec(cleanNoteID, n.Note, n.NoteFlesh, n.Type, n.Status, n.Area, n.Importance, n.Clarity, n.Source, targetDT, n.CreatedAt, n.UpdatedAt, deletedDT, n.DeletedNote); err != nil {
				return err
			}

			if _, err := stmtFtsDel.Exec(cleanNoteID); err != nil {
				return err
			}
			if n.DeletedAt.IsZero() {
				if _, err := stmtFtsIns.Exec(cleanNoteID, n.Note, n.NoteFlesh); err != nil {
					return err
				}
			}
		}
	}

	// 3. Clear Note Tags for Updated/Pulled Notes
	if len(data.ClearedNotes) > 0 {
		stmtClearNT, err := tx.Prepare("DELETE FROM note_tags WHERE note_id = ?")
		if err != nil {
			return err
		}
		defer stmtClearNT.Close()

		for _, noteID := range data.ClearedNotes {
			cleanNoteID := strings.ReplaceAll(noteID, "-", "")
			if _, err := stmtClearNT.Exec(cleanNoteID); err != nil {
				return err
			}
		}
	}

	// 4. Process Note Tags
	if len(data.NoteTags) > 0 {
		stmtNoteTag, err := tx.Prepare("INSERT OR IGNORE INTO note_tags (note_id, tag_id, created_at) VALUES (?, ?, ?)")
		if err != nil {
			return err
		}
		defer stmtNoteTag.Close()

		for _, nt := range data.NoteTags {
			cleanNoteID := strings.ReplaceAll(nt.NoteID, "-", "")
			if _, err := stmtNoteTag.Exec(cleanNoteID, nt.TagID, nt.CreatedAt); err != nil {
				return err
			}
		}
	}

	// 5. Process Links
	if len(data.Links) > 0 {
		linkQuery := `
			INSERT INTO links (id, from_note, to_note, type, created_at, deleted_at, deleted_note)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				from_note = excluded.from_note,
				to_note = excluded.to_note,
				type = excluded.type,
				created_at = excluded.created_at,
				deleted_at = excluded.deleted_at,
				deleted_note = excluded.deleted_note
			WHERE (excluded.deleted_at IS NOT NULL AND (links.deleted_at IS NULL OR excluded.deleted_at >= links.deleted_at))
			   OR (links.deleted_at IS NULL)
		`
		stmtLink, err := tx.Prepare(linkQuery)
		if err != nil {
			return err
		}
		defer stmtLink.Close()

		for _, l := range data.Links {
			cleanLinkID := strings.ReplaceAll(l.ID, "-", "")
			cleanFrom := strings.ReplaceAll(l.FromNote, "-", "")
			cleanTo := strings.ReplaceAll(l.ToNote, "-", "")
			var deletedDT sql.NullTime
			if !l.DeletedAt.IsZero() {
				deletedDT = sql.NullTime{Time: l.DeletedAt, Valid: true}
			}
			if _, err := stmtLink.Exec(cleanLinkID, cleanFrom, cleanTo, l.Type, l.CreatedAt, deletedDT, l.DeletedNote); err != nil {
				return err
			}
		}
	}

	// 6. Process Daily Logs
	if len(data.DailyLogs) > 0 {
		logQuery := `
			INSERT INTO daily_logs (id, content, note_id, created_at, deleted_at, deleted_note)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				content = excluded.content,
				note_id = excluded.note_id,
				created_at = excluded.created_at,
				deleted_at = excluded.deleted_at,
				deleted_note = excluded.deleted_note
		`
		stmtLog, err := tx.Prepare(logQuery)
		if err != nil {
			return err
		}
		defer stmtLog.Close()

		for _, l := range data.DailyLogs {
			cleanLogID := strings.ReplaceAll(l.ID, "-", "")
			var noteID sql.NullString
			if l.NoteID != "" {
				noteID = sql.NullString{String: strings.ReplaceAll(l.NoteID, "-", ""), Valid: true}
			}
			var deletedDT sql.NullTime
			if !l.DeletedAt.IsZero() {
				deletedDT = sql.NullTime{Time: l.DeletedAt, Valid: true}
			}
			if _, err := stmtLog.Exec(cleanLogID, l.Content, noteID, l.CreatedAt, deletedDT, l.DeletedNote); err != nil {
				return err
			}
		}
	}

	// 7. Process Audit Logs
	if len(data.AuditLogs) > 0 {
		stmtAudit, err := tx.Prepare(`
			INSERT INTO audit_logs (id, entity_type, entity_id, action, changes_summary, snapshot_json, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO NOTHING
		`)
		if err != nil {
			return err
		}
		defer stmtAudit.Close()

		for _, a := range data.AuditLogs {
			cleanAuditID := strings.ReplaceAll(a.ID, "-", "")
			cleanEntityID := strings.ReplaceAll(a.EntityID, "-", "")
			if _, err := stmtAudit.Exec(cleanAuditID, a.EntityType, cleanEntityID, string(a.Action), a.ChangesSummary, a.SnapshotJSON, a.CreatedAt); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

