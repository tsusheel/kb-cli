package db

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/tsusheel/kb-cli/models"
)

func AddLink(fromID string, toID string, linkType models.LinkType) error {
	fullFromID, err := ResolveID(fromID)
	if err != nil {
		return err
	}

	fullToID, err := ResolveID(toID)
	if err != nil {
		return err
	}

	linkID := uuid.New().String()
	now := time.Now()
	query := `INSERT INTO links (id, from_note, to_note, type, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err = DB.Exec(query, linkID, fullFromID, fullToID, linkType, now)
	return err
}

func SoftDeleteLink(linkID string, reason string) error {
	if reason == "" {
		reason = "deleted by AI"
	}
	now := time.Now()
	query := `UPDATE links SET deleted_at = ?, deleted_note = ? WHERE id = ?`
	_, err := DB.Exec(query, now, reason, linkID)
	return err
}

func GetLinksForNote(noteID string) ([]models.Link, error) {
	fullNoteID, err := ResolveID(noteID)
	if err != nil {
		return nil, err
	}

	query := `SELECT id, from_note, to_note, type, created_at, deleted_at, deleted_note FROM links WHERE (from_note = ? OR to_note = ?) AND deleted_at IS NULL`
	rows, err := DB.Query(query, fullNoteID, fullNoteID)
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
