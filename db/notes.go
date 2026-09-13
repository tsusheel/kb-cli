package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
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

	var targetDT sql.NullTime
	if !n.TargetDateTime.IsZero() {
		targetDT = sql.NullTime{Time: n.TargetDateTime, Valid: true}
	}

	query := `INSERT INTO notes (
		id, note, note_flesh, type, status, area, importance, clarity, source, target_date_time, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = tx.Exec(query, n.ID, n.Note, n.NoteFlesh, n.Type, n.Status, n.Area, n.Importance, n.Clarity, n.Source, targetDT, n.CreatedAt, n.UpdatedAt)
	if err != nil {
		return err
	}

	ftsQuery := `INSERT INTO notes_fts (note_id, note, note_flesh) VALUES (?, ?, ?)`
	_, err = tx.Exec(ftsQuery, n.ID, n.Note, n.NoteFlesh)
	if err != nil {
		return err
	}

	// Record audit log
	if err := RecordAudit(tx, "note", n.ID, models.ActionCreated, "created note", n); err != nil {
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

	query := `SELECT id, note, note_flesh, type, status, area, importance, clarity, source, target_date_time, created_at, updated_at, deleted_at, deleted_note FROM notes WHERE id = ?`
	row := DB.QueryRow(query, fullID)

	var n models.Note
	var targetDT sql.NullTime
	var deletedDT sql.NullTime
	var deletedNote sql.NullString
	err = row.Scan(&n.ID, &n.Note, &n.NoteFlesh, &n.Type, &n.Status, &n.Area, &n.Importance, &n.Clarity, &n.Source, &targetDT, &n.CreatedAt, &n.UpdatedAt, &deletedDT, &deletedNote)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
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

	return &n, nil
}

func UpdateNote(n *models.Note) error {
	fullID, err := ResolveID(n.ID)
	if err != nil {
		return err
	}
	n.ID = fullID

	oldNote, err := GetNote(n.ID)
	var diffSummary string
	if err == nil {
		diffSummary = ComputeNoteDiffSummary(oldNote, n)
	} else {
		diffSummary = "updated note"
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	n.UpdatedAt = time.Now()

	var targetDT sql.NullTime
	if !n.TargetDateTime.IsZero() {
		targetDT = sql.NullTime{Time: n.TargetDateTime, Valid: true}
	}

	query := `UPDATE notes SET 
		note = ?, 
		note_flesh = ?, 
		type = ?, 
		status = ?, 
		area = ?, 
		importance = ?, 
		clarity = ?, 
		source = ?, 
		target_date_time = ?, 
		updated_at = ?
	WHERE id = ?`

	_, err = tx.Exec(query, n.Note, n.NoteFlesh, n.Type, n.Status, n.Area, n.Importance, n.Clarity, n.Source, targetDT, n.UpdatedAt, n.ID)
	if err != nil {
		return err
	}

	// Update FTS table
	_, err = tx.Exec(`DELETE FROM notes_fts WHERE note_id = ?`, n.ID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO notes_fts (note_id, note, note_flesh) VALUES (?, ?, ?)`, n.ID, n.Note, n.NoteFlesh)
	if err != nil {
		return err
	}

	// Record audit log
	if err := RecordAudit(tx, "note", n.ID, models.ActionUpdated, diffSummary, n); err != nil {
		return err
	}

	return tx.Commit()
}

func SoftDeleteNote(id string, reason string) error {
	fullID, err := ResolveID(id)
	if err != nil {
		return err
	}

	if reason == "" {
		reason = "deleted"
	}

	n, _ := GetNote(fullID)

	now := time.Now()
	query := `UPDATE notes SET deleted_at = ?, deleted_note = ?, updated_at = ? WHERE id = ?`
	_, err = DB.Exec(query, now, reason, now, fullID)
	if err != nil {
		return err
	}

	if n != nil {
		n.DeletedAt = now
		n.DeletedNote = reason
		n.UpdatedAt = now
		_ = RecordAudit(nil, "note", fullID, models.ActionDeleted, fmt.Sprintf("soft-deleted: %s", reason), n)
	}

	return nil
}

func RestoreNote(id string) (*models.Note, error) {
	fullID, err := ResolveID(id)
	if err != nil {
		return nil, err
	}

	n, err := GetNote(fullID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	query := `UPDATE notes SET deleted_at = NULL, deleted_note = NULL, updated_at = ? WHERE id = ?`
	if _, err := DB.Exec(query, now, fullID); err != nil {
		return nil, err
	}

	n.DeletedAt = time.Time{}
	n.DeletedNote = ""
	n.UpdatedAt = now
	_ = RecordAudit(nil, "note", fullID, models.ActionRestored, "restored note", n)

	return n, nil
}

func ListNotes(filterType string) ([]models.Note, error) {
	return ListNotesExtended(filterType, "", "", false)
}

func ListNotesExtended(filterType, filterStatus, filterArea string, includeDeleted bool) ([]models.Note, error) {
	var whereClauses []string
	var args []interface{}

	if !includeDeleted {
		whereClauses = append(whereClauses, "deleted_at IS NULL")
	}
	if filterType != "" {
		whereClauses = append(whereClauses, "type = ?")
		args = append(args, filterType)
	}
	if filterStatus != "" {
		whereClauses = append(whereClauses, "status = ?")
		args = append(args, filterStatus)
	}
	if filterArea != "" {
		whereClauses = append(whereClauses, "area = ?")
		args = append(args, filterArea)
	}

	query := "SELECT id, note, type, status, area, target_date_time, created_at, updated_at, deleted_at, deleted_note FROM notes"
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}
	query += " ORDER BY updated_at DESC"

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
		err := rows.Scan(&n.ID, &n.Note, &n.Type, &n.Status, &n.Area, &targetDT, &n.CreatedAt, &n.UpdatedAt, &deletedDT, &deletedNote)
		if err != nil {
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

func SearchNotesExtended(searchTerm, filterType, filterStatus, filterArea string) ([]models.Note, error) {
	whereClauses := []string{"notes_fts MATCH ?", "n.deleted_at IS NULL"}
	args := []interface{}{searchTerm}

	if filterType != "" {
		whereClauses = append(whereClauses, "n.type = ?")
		args = append(args, filterType)
	}
	if filterStatus != "" {
		whereClauses = append(whereClauses, "n.status = ?")
		args = append(args, filterStatus)
	}
	if filterArea != "" {
		whereClauses = append(whereClauses, "n.area = ?")
		args = append(args, filterArea)
	}

	query := fmt.Sprintf(`
		SELECT n.id, n.note, n.type, n.status, n.area, n.target_date_time, n.created_at, n.updated_at 
		FROM notes_fts fts
		JOIN notes n ON n.id = fts.note_id
		WHERE %s
		ORDER BY rank
	`, strings.Join(whereClauses, " AND "))

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		var n models.Note
		var targetDT sql.NullTime
		err := rows.Scan(&n.ID, &n.Note, &n.Type, &n.Status, &n.Area, &targetDT, &n.CreatedAt, &n.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if targetDT.Valid {
			n.TargetDateTime = targetDT.Time
		}
		notes = append(notes, n)
	}

	return notes, nil
}

func SearchNotes(searchTerm string) ([]models.Note, error) {
	return SearchNotesExtended(searchTerm, "", "", "")
}


// GetRawNotes returns raw unrefined notes for today (todayOnly=true) or all time (todayOnly=false).
func GetRawNotes(todayOnly bool) ([]models.Note, error) {
	var query string
	var args []interface{}

	if todayOnly {
		now := time.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endOfDay := startOfDay.AddDate(0, 0, 1)

		query = `SELECT id, note, note_flesh, type, status, area, target_date_time, created_at, updated_at, deleted_at, deleted_note 
		         FROM notes 
		         WHERE status = 'raw' AND deleted_at IS NULL AND created_at >= ? AND created_at < ? 
		         ORDER BY updated_at DESC`
		args = []interface{}{startOfDay, endOfDay}
	} else {
		query = `SELECT id, note, note_flesh, type, status, area, target_date_time, created_at, updated_at, deleted_at, deleted_note 
		         FROM notes 
		         WHERE status = 'raw' AND deleted_at IS NULL 
		         ORDER BY updated_at DESC`
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
		err := rows.Scan(&n.ID, &n.Note, &n.NoteFlesh, &n.Type, &n.Status, &n.Area, &targetDT, &n.CreatedAt, &n.UpdatedAt, &deletedDT, &deletedNote)
		if err != nil {
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
