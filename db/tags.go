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
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}

// GetAllNoteTagsMap fetches a map of Note ID -> slice of Tag Names in a single efficient SQL query.
func GetAllNoteTagsMap() (map[string][]string, error) {
	query := `
		SELECT nt.note_id, t.name 
		FROM note_tags nt
		JOIN tags t ON nt.tag_id = t.id
		ORDER BY nt.created_at ASC
	`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tagsMap := make(map[string][]string)
	for rows.Next() {
		var noteID, tagName string
		if err := rows.Scan(&noteID, &tagName); err != nil {
			return nil, err
		}
		tagsMap[noteID] = append(tagsMap[noteID], tagName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tagsMap, nil
}

func RemoveTag(noteID string, tagName string) error {
	fullNoteID, err := ResolveID(noteID)
	if err != nil {
		return err
	}

	query := `
		DELETE FROM note_tags 
		WHERE note_id = ? AND tag_id IN (SELECT id FROM tags WHERE name = ?)
	`
	_, err = DB.Exec(query, fullNoteID, tagName)
	return err
}

