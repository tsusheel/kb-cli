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

	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `INSERT INTO daily_logs (id, content, created_at) VALUES (?, ?, ?)`
	_, err = tx.Exec(query, id, content, now)
	if err != nil {
		return nil, err
	}

	logEntry := &models.DailyLog{
		ID:        id,
		Content:   content,
		CreatedAt: now,
	}

	if err := RecordAudit(tx, "daily_log", id, models.ActionCreated, "created daily log", logEntry); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return logEntry, nil
}

func UpdateDailyLog(l *models.DailyLog) error {
	fullID, err := ResolveLogID(l.ID)
	if err != nil {
		return err
	}

	l.Content = strings.TrimSpace(l.Content)
	if l.Content == "" {
		return fmt.Errorf("daily log content cannot be empty")
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `UPDATE daily_logs SET content = ? WHERE id = ?`
	_, err = tx.Exec(query, l.Content, fullID)
	if err != nil {
		return err
	}

	if err := RecordAudit(tx, "daily_log", fullID, models.ActionUpdated, "content updated", l); err != nil {
		return err
	}

	return tx.Commit()
}


type logRowScanner interface {
	Scan(dest ...interface{}) error
}

const logColumns = "id, content, note_id, created_at, deleted_at, deleted_note"

func scanLog(s logRowScanner) (*models.DailyLog, error) {
	var l models.DailyLog
	var noteID sql.NullString
	var deletedAt sql.NullTime
	var deletedNote sql.NullString

	if err := s.Scan(&l.ID, &l.Content, &noteID, &l.CreatedAt, &deletedAt, &deletedNote); err != nil {
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

func ResolveLogID(id string) (string, error) {
	cleanID := strings.ReplaceAll(id, "-", "")
	if len(cleanID) == 32 {
		var exists string
		err := DB.QueryRow("SELECT id FROM daily_logs WHERE id = ?", cleanID).Scan(&exists)
		if err == sql.ErrNoRows {
			return "", ErrLogNotFound
		} else if err != nil {
			return "", err
		}
		return cleanID, nil
	}

	query := `SELECT id FROM daily_logs WHERE id LIKE ?`
	rows, err := DB.Query(query, cleanID+"%")
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
	if err := rows.Err(); err != nil {
		return "", err
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

	query := `SELECT ` + logColumns + ` FROM daily_logs WHERE id = ?`
	row := DB.QueryRow(query, fullID)
	return scanLog(row)
}

// GetDailyLogsFilter queries daily logs with optional date range, promotion status, and soft-delete filters.
func GetDailyLogsFilter(startDate, endDate *time.Time, includePromoted bool, includeDeleted bool) ([]models.DailyLog, error) {
	var whereClauses []string
	var args []interface{}

	if !includeDeleted {
		whereClauses = append(whereClauses, "deleted_at IS NULL")
	}

	if !includePromoted {
		whereClauses = append(whereClauses, "(note_id IS NULL OR note_id = '')")
	}

	if startDate != nil {
		startOfDay := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
		whereClauses = append(whereClauses, "created_at >= ?")
		args = append(args, startOfDay)
	}

	if endDate != nil {
		endOfDay := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location()).AddDate(0, 0, 1)
		whereClauses = append(whereClauses, "created_at < ?")
		args = append(args, endOfDay)
	}

	query := `SELECT ` + logColumns + ` FROM daily_logs`
	if len(whereClauses) > 0 {
		query += ` WHERE ` + strings.Join(whereClauses, " AND ")
	}
	query += ` ORDER BY created_at ASC`

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

func GetDailyLogsForDate(date time.Time, includePromoted bool) ([]models.DailyLog, error) {
	return GetDailyLogsFilter(&date, &date, includePromoted, false)
}

func GetUnpromotedDailyLogs() ([]models.DailyLog, error) {
	query := `SELECT ` + logColumns + ` FROM daily_logs WHERE (note_id IS NULL OR note_id = '') AND deleted_at IS NULL ORDER BY created_at DESC`
	rows, err := DB.Query(query)
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
		noteStatus = models.Active
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

	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Update daily_logs table with note_id reference
	query := `UPDATE daily_logs SET note_id = ? WHERE id = ?`
	if _, err := tx.Exec(query, n.ID, l.ID); err != nil {
		return nil, fmt.Errorf("failed to link log to note: %w", err)
	}

	l.NoteID = n.ID
	if err := RecordAudit(tx, "daily_log", l.ID, models.ActionPromoted, fmt.Sprintf("promoted to note [%s]", n.ID[:7]), l); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
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

	l, _ := GetDailyLog(fullID)

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	query := `UPDATE daily_logs SET deleted_at = ?, deleted_note = ? WHERE id = ?`
	_, err = tx.Exec(query, now, reason, fullID)
	if err != nil {
		return err
	}

	if l != nil {
		l.DeletedAt = now
		l.DeletedNote = reason
		if err := RecordAudit(tx, "daily_log", fullID, models.ActionDeleted, fmt.Sprintf("soft-deleted: %s", reason), l); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func RestoreDailyLog(id string) (*models.DailyLog, error) {
	fullID, err := ResolveLogID(id)
	if err != nil {
		return nil, err
	}

	l, err := GetDailyLog(fullID)
	if err != nil {
		return nil, err
	}

	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `UPDATE daily_logs SET deleted_at = NULL, deleted_note = NULL WHERE id = ?`
	if _, err := tx.Exec(query, fullID); err != nil {
		return nil, err
	}

	l.DeletedAt = time.Time{}
	l.DeletedNote = ""
	if err := RecordAudit(tx, "daily_log", fullID, models.ActionRestored, "restored daily log", l); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return l, nil
}
