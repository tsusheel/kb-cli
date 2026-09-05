package db

import (
	"database/sql"
	"time"

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

	now := time.Now()

	// Find or Create Tag
	var tagID string
	err = tx.QueryRow("SELECT id FROM tags WHERE name = ?", tagName).Scan(&tagID)
	if err != nil {
		if err == sql.ErrNoRows {
			tagID = uuid.New().String()
			_, err = tx.Exec("INSERT INTO tags (id, name, created_at) VALUES (?, ?, ?)", tagID, tagName, now)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// Link tag to note
	_, err = tx.Exec("INSERT OR IGNORE INTO note_tags (note_id, tag_id, created_at) VALUES (?, ?, ?)", fullNoteID, tagID, now)
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
		SELECT t.id, t.name, t.created_at 
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
