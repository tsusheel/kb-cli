package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tsusheel/kb-cli/models"
)

var ErrLogNotFound = errors.New("daily log not found")
var ErrLogAmbiguous = errors.New("ambiguous short id, multiple logs found")

func CreateDailyLog(content string) (*models.DailyLog, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("log content cannot be empty")
	}

	id := strings.ReplaceAll(uuid.New().String(), "-", "")
	now := time.Now()

	query := `INSERT INTO daily_logs (id, content, created_at) VALUES (?, ?, ?)`
	_, err := DB.Exec(query, id, content, now)
	if err != nil {
		return nil, err
	}

	return &models.DailyLog{
		ID:        id,
		Content:   content,
		CreatedAt: now,
	}, nil
}

func ResolveLogID(id string) (string, error) {
	if len(id) == 32 || len(id) == 36 {
		return strings.ReplaceAll(id, "-", ""), nil
	}

	query := `SELECT id FROM daily_logs WHERE id LIKE ?`
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
		return "", ErrLogNotFound
	}
	if count > 1 {
		return "", ErrLogAmbiguous
	}

	return matchedID, nil
}

func GetDailyLog(id string) (*models.DailyLog, error) {
	fullID, err := ResolveLogID(id)
	if err != nil {
		return nil, err
	}

	query := `SELECT id, content, note_id, created_at, deleted_at, deleted_note FROM daily_logs WHERE id = ?`
	row := DB.QueryRow(query, fullID)

	var l models.DailyLog
	var noteID sql.NullString
	var deletedAt sql.NullTime
	var deletedNote sql.NullString

	err = row.Scan(&l.ID, &l.Content, &noteID, &l.CreatedAt, &deletedAt, &deletedNote)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrLogNotFound
		}
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

	return &l, nil
}

func GetDailyLogsForDate(date time.Time, includePromoted bool) ([]models.DailyLog, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.AddDate(0, 0, 1)

	var query string
	var args []interface{}

	if includePromoted {
		query = `SELECT id, content, note_id, created_at, deleted_at, deleted_note FROM daily_logs WHERE created_at >= ? AND created_at < ? AND deleted_at IS NULL ORDER BY created_at ASC`
		args = []interface{}{startOfDay, endOfDay}
	} else {
		query = `SELECT id, content, note_id, created_at, deleted_at, deleted_note FROM daily_logs WHERE created_at >= ? AND created_at < ? AND (note_id IS NULL OR note_id = '') AND deleted_at IS NULL ORDER BY created_at ASC`
		args = []interface{}{startOfDay, endOfDay}
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

func GetUnpromotedDailyLogs() ([]models.DailyLog, error) {
	query := `SELECT id, content, note_id, created_at, deleted_at, deleted_note FROM daily_logs WHERE (note_id IS NULL OR note_id = '') AND deleted_at IS NULL ORDER BY created_at DESC`
	rows, err := DB.Query(query)
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

func PromoteDailyLog(logID string, noteType models.NoteType, noteStatus models.Status) (*models.Note, error) {
	l, err := GetDailyLog(logID)
	if err != nil {
		return nil, err
	}

	if l.NoteID != "" {
		return nil, fmt.Errorf("log [%s] is already promoted to note [%s]", l.ID[:7], l.NoteID[:7])
	}

	if noteType == "" {
		noteType = models.DefaultNote
	}
	if noteStatus == "" {
		noteStatus = models.Raw
	}

	noteID := strings.ReplaceAll(uuid.New().String(), "-", "")
	now := time.Now()

	n := &models.Note{
		ID:        noteID,
		Note:      l.Content,
		NoteFlesh: "",
		Type:      noteType,
		Status:    noteStatus,
		Source:    "Daily Log",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := CreateNote(n); err != nil {
		return nil, fmt.Errorf("failed to create note from log: %w", err)
	}

	// Update daily_logs table with note_id reference
	query := `UPDATE daily_logs SET note_id = ? WHERE id = ?`
	if _, err := DB.Exec(query, n.ID, l.ID); err != nil {
		return nil, fmt.Errorf("failed to link log to note: %w", err)
	}

	return n, nil
}

func SoftDeleteDailyLog(id string, reason string) error {
	fullID, err := ResolveLogID(id)
	if err != nil {
		return err
	}

	if reason == "" {
		reason = "deleted"
	}

	now := time.Now()
	query := `UPDATE daily_logs SET deleted_at = ?, deleted_note = ? WHERE id = ?`
	_, err = DB.Exec(query, now, reason, fullID)
	return err
}
