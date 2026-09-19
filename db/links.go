package db

import (
	"database/sql"
	"fmt"
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
		reason = "deleted"
	}
	now := time.Now()
	query := `UPDATE links SET deleted_at = ?, deleted_note = ? WHERE id = ?`
	_, err := DB.Exec(query, now, reason, linkID)
	return err
}

func RemoveLink(fromID, toID string, reason string) error {
	fullFromID, err := ResolveID(fromID)
	if err != nil {
		return err
	}
	fullToID, err := ResolveID(toID)
	if err != nil {
		return err
	}
	if reason == "" {
		reason = "deleted"
	}
	now := time.Now()
	query := `UPDATE links SET deleted_at = ?, deleted_note = ? WHERE from_note = ? AND to_note = ? AND deleted_at IS NULL`
	res, err := DB.Exec(query, now, reason, fullFromID, fullToID)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		fromDisplay := fullFromID
		if len(fromDisplay) > 7 {
			fromDisplay = fromDisplay[:7]
		}
		toDisplay := fullToID
		if len(toDisplay) > 7 {
			toDisplay = toDisplay[:7]
		}
		return fmt.Errorf("no active link found from [%s] to [%s]", fromDisplay, toDisplay)
	}
	return nil
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

func GetIncomingLinks(noteID string) ([]models.Link, error) {
	fullNoteID, err := ResolveID(noteID)
	if err != nil {
		return nil, err
	}

	query := `SELECT id, from_note, to_note, type, created_at, deleted_at, deleted_note FROM links WHERE to_note = ? AND deleted_at IS NULL ORDER BY created_at DESC`
	rows, err := DB.Query(query, fullNoteID)
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

func GetOutgoingLinks(noteID string) ([]models.Link, error) {
	fullNoteID, err := ResolveID(noteID)
	if err != nil {
		return nil, err
	}

	query := `SELECT id, from_note, to_note, type, created_at, deleted_at, deleted_note FROM links WHERE from_note = ? AND deleted_at IS NULL ORDER BY created_at DESC`
	rows, err := DB.Query(query, fullNoteID)
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

func GetAllActiveLinks() ([]models.Link, error) {
	query := `SELECT id, from_note, to_note, type, created_at, deleted_at, deleted_note FROM links WHERE deleted_at IS NULL`
	rows, err := DB.Query(query)
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
