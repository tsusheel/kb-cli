package db

import (
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
	query := `INSERT INTO links (id, from_note, to_note, type) VALUES (?, ?, ?, ?)`
	_, err = DB.Exec(query, linkID, fullFromID, fullToID, linkType)
	return err
}

func GetLinksForNote(noteID string) ([]models.Link, error) {
	fullNoteID, err := ResolveID(noteID)
	if err != nil {
		return nil, err
	}

	query := `SELECT id, from_note, to_note, type FROM links WHERE from_note = ? OR to_note = ?`
	rows, err := DB.Query(query, fullNoteID, fullNoteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.Link
	for rows.Next() {
		var l models.Link
		if err := rows.Scan(&l.ID, &l.FromNote, &l.ToNote, &l.Type); err != nil {
			return nil, err
		}
		links = append(links, l)
	}

	return links, nil
}
