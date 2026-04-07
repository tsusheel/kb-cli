package db

import (
	"database/sql"
	"errors"
	"time"

	"github.com/tsusheel/kb-cli/models"
)

var ErrNotFound = errors.New("note not found")
var ErrAmbiguous = errors.New("ambiguous short id, multiple notes found")

func CreateNote(n *models.Note) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}
	if n.UpdatedAt.IsZero() {
		n.UpdatedAt = time.Now()
	}

	query := `INSERT INTO notes (
		id, title, content, type, status, area, importance, clarity, source, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	_, err = tx.Exec(query, n.ID, n.Title, n.Content, n.Type, n.Status, n.Area, n.Importance, n.Clarity, n.Source, n.CreatedAt, n.UpdatedAt)
	if err != nil {
		return err
	}

	ftsQuery := `INSERT INTO notes_fts (note_id, title, content) VALUES (?, ?, ?)`
	_, err = tx.Exec(ftsQuery, n.ID, n.Title, n.Content)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func ResolveID(id string) (string, error) {
	if len(id) == 36 { // full UUID
		return id, nil
	}
	
	query := `SELECT id FROM notes WHERE id LIKE ?`
	rows, err := DB.Query(query, id+"%")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var matchedID string
	count := 0
	for rows.Next() {
		count++
		if err := rows.Scan(&matchedID); err != nil {
			return "", err
		}
	}
	
	if count == 0 {
		return "", ErrNotFound
	}
	if count > 1 {
		return "", ErrAmbiguous
	}
	
	return matchedID, nil
}

func GetNote(id string) (*models.Note, error) {
	fullID, err := ResolveID(id)
	if err != nil {
		return nil, err
	}

	query := `SELECT id, title, content, type, status, area, importance, clarity, source, created_at, updated_at FROM notes WHERE id = ?`
	row := DB.QueryRow(query, fullID)

	var n models.Note
	err = row.Scan(&n.ID, &n.Title, &n.Content, &n.Type, &n.Status, &n.Area, &n.Importance, &n.Clarity, &n.Source, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &n, nil
}

func ListNotes(filterType string) ([]models.Note, error) {
	var query string
	var args []interface{}

	if filterType != "" {
		query = `SELECT id, title, type, status, area, created_at, updated_at FROM notes WHERE type = ? ORDER BY updated_at DESC`
		args = append(args, filterType)
	} else {
		query = `SELECT id, title, type, status, area, created_at, updated_at FROM notes ORDER BY updated_at DESC`
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		var n models.Note
		err := rows.Scan(&n.ID, &n.Title, &n.Type, &n.Status, &n.Area, &n.CreatedAt, &n.UpdatedAt)
		if err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}

	return notes, nil
}

func SearchNotes(searchTerm string) ([]models.Note, error) {
	query := `
		SELECT n.id, n.title, n.type, n.status, n.area, n.created_at, n.updated_at 
		FROM notes_fts fts
		JOIN notes n ON n.id = fts.note_id
		WHERE notes_fts MATCH ?
		ORDER BY rank
	`
	rows, err := DB.Query(query, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		var n models.Note
		err := rows.Scan(&n.ID, &n.Title, &n.Type, &n.Status, &n.Area, &n.CreatedAt, &n.UpdatedAt)
		if err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}

	return notes, nil
}
