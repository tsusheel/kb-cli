package db

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/tsusheel/kb-cli/models"
)

func AddTag(noteID string, tagName string) error {
	fullNoteID, err := ResolveID(noteID)
	if err != nil {
		return err
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Find or Create Tag
	var tagID string
	err = tx.QueryRow("SELECT id FROM tags WHERE name = ?", tagName).Scan(&tagID)
	if err != nil {
		if err == sql.ErrNoRows {
			tagID = uuid.New().String()
			_, err = tx.Exec("INSERT INTO tags (id, name) VALUES (?, ?)", tagID, tagName)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// Link tag to note
	_, err = tx.Exec("INSERT OR IGNORE INTO note_tags (note_id, tag_id) VALUES (?, ?)", fullNoteID, tagID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func GetTagsForNote(noteID string) ([]models.Tag, error) {
	fullNoteID, err := ResolveID(noteID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT t.id, t.name 
		FROM tags t
		JOIN note_tags nt ON t.id = nt.tag_id
		WHERE nt.note_id = ?
	`
	rows, err := DB.Query(query, fullNoteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}

	return tags, nil
}
