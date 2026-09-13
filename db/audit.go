package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tsusheel/kb-cli/models"
)

var ErrAuditNotFound = errors.New("audit log not found")
var ErrAuditAmbiguous = errors.New("ambiguous short id, multiple audit logs found")

// RecordAudit records an audit revision entry for an entity (inside or outside a transaction).
func RecordAudit(tx *sql.Tx, entityType, entityID string, action models.AuditAction, summary string, snapshot interface{}) error {
	id := strings.ReplaceAll(uuid.New().String(), "-", "")
	entityID = strings.ReplaceAll(entityID, "-", "")
	now := time.Now()

	var snapshotJSON string
	if snapshot != nil {
		data, err := json.Marshal(snapshot)
		if err == nil {
			snapshotJSON = string(data)
		}
	}

	query := `INSERT INTO audit_logs (id, entity_type, entity_id, action, changes_summary, snapshot_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	if tx != nil {
		_, err := tx.Exec(query, id, entityType, entityID, string(action), summary, snapshotJSON, now)
		return err
	}

	_, err := DB.Exec(query, id, entityType, entityID, string(action), summary, snapshotJSON, now)
	return err
}

// ComputeNoteDiffSummary compares two notes and produces a concise human-readable summary of the changes.
func ComputeNoteDiffSummary(oldNote, newNote *models.Note) string {
	if oldNote == nil || newNote == nil {
		return "updated"
	}

	var changes []string

	if oldNote.Note != newNote.Note {
		if len(oldNote.Note) < 20 && len(newNote.Note) < 20 {
			changes = append(changes, fmt.Sprintf("title: %q -> %q", oldNote.Note, newNote.Note))
		} else {
			changes = append(changes, "title updated")
		}
	}

	if oldNote.Status != newNote.Status && newNote.Status != "" {
		changes = append(changes, fmt.Sprintf("status: %s -> %s", oldNote.Status, newNote.Status))
	}

	if oldNote.Type != newNote.Type && newNote.Type != "" {
		changes = append(changes, fmt.Sprintf("type: %s -> %s", oldNote.Type, newNote.Type))
	}

	if oldNote.Area != newNote.Area {
		oldArea := string(oldNote.Area)
		if oldArea == "" {
			oldArea = "none"
		}
		newArea := string(newNote.Area)
		if newArea == "" {
			newArea = "none"
		}
		changes = append(changes, fmt.Sprintf("area: %s -> %s", oldArea, newArea))
	}

	if oldNote.TargetDateTime != newNote.TargetDateTime {
		if newNote.TargetDateTime.IsZero() {
			changes = append(changes, "due date cleared")
		} else if oldNote.TargetDateTime.IsZero() {
			changes = append(changes, fmt.Sprintf("due: %s", newNote.TargetDateTime.Format("2006-01-02 15:04")))
		} else {
			changes = append(changes, fmt.Sprintf("due: %s -> %s", oldNote.TargetDateTime.Format("2006-01-02"), newNote.TargetDateTime.Format("2006-01-02")))
		}
	}

	if oldNote.NoteFlesh != newNote.NoteFlesh {
		charDiff := len(newNote.NoteFlesh) - len(oldNote.NoteFlesh)
		if oldNote.NoteFlesh == "" && newNote.NoteFlesh != "" {
			changes = append(changes, fmt.Sprintf("body added (%d chars)", len(newNote.NoteFlesh)))
		} else if newNote.NoteFlesh == "" && oldNote.NoteFlesh != "" {
			changes = append(changes, "body cleared")
		} else if charDiff > 0 {
			changes = append(changes, fmt.Sprintf("body edited (+%d chars)", charDiff))
		} else if charDiff < 0 {
			changes = append(changes, fmt.Sprintf("body edited (%d chars)", charDiff))
		} else {
			changes = append(changes, "body edited")
		}
	}

	if oldNote.Importance != newNote.Importance && newNote.Importance > 0 {
		changes = append(changes, fmt.Sprintf("importance: %d -> %d", oldNote.Importance, newNote.Importance))
	}

	if oldNote.Clarity != newNote.Clarity && newNote.Clarity > 0 {
		changes = append(changes, fmt.Sprintf("clarity: %d -> %d", oldNote.Clarity, newNote.Clarity))
	}

	if len(changes) == 0 {
		return "metadata updated"
	}

	return strings.Join(changes, ", ")
}

// GetAuditHistory retrieves the revision timeline for a specific entity.
func GetAuditHistory(entityType, entityID string, limit int) ([]models.AuditEntry, error) {
	entityID = strings.ReplaceAll(entityID, "-", "")
	if limit <= 0 {
		limit = 50
	}

	query := `SELECT id, entity_type, entity_id, action, changes_summary, snapshot_json, created_at
		FROM audit_logs
		WHERE entity_type = ? AND entity_id = ?
		ORDER BY created_at DESC, rowid DESC
		LIMIT ?`

	rows, err := DB.Query(query, entityType, entityID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.AuditEntry
	for rows.Next() {
		var entry models.AuditEntry
		var summary sql.NullString
		var snapshot sql.NullString
		var action string

		if err := rows.Scan(&entry.ID, &entry.EntityType, &entry.EntityID, &action, &summary, &snapshot, &entry.CreatedAt); err != nil {
			return nil, err
		}
		entry.Action = models.AuditAction(action)
		if summary.Valid {
			entry.ChangesSummary = summary.String
		}
		if snapshot.Valid {
			entry.SnapshotJSON = snapshot.String
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// GetRecentAuditHistory retrieves global recent audit logs across all entities.
func GetRecentAuditHistory(limit int) ([]models.AuditEntry, error) {
	if limit <= 0 {
		limit = 30
	}

	query := `SELECT id, entity_type, entity_id, action, changes_summary, snapshot_json, created_at
		FROM audit_logs
		ORDER BY created_at DESC, rowid DESC
		LIMIT ?`

	rows, err := DB.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.AuditEntry
	for rows.Next() {
		var entry models.AuditEntry
		var summary sql.NullString
		var snapshot sql.NullString
		var action string

		if err := rows.Scan(&entry.ID, &entry.EntityType, &entry.EntityID, &action, &summary, &snapshot, &entry.CreatedAt); err != nil {
			return nil, err
		}
		entry.Action = models.AuditAction(action)
		if summary.Valid {
			entry.ChangesSummary = summary.String
		}
		if snapshot.Valid {
			entry.SnapshotJSON = snapshot.String
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// ResolveAuditID resolves short 7+ char prefixes to full 32-char UUIDs.
func ResolveAuditID(id string) (string, error) {
	cleanID := strings.ReplaceAll(id, "-", "")
	if len(cleanID) == 32 {
		return cleanID, nil
	}

	query := `SELECT id FROM audit_logs WHERE id LIKE ?`
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

	if count == 0 {
		return "", ErrAuditNotFound
	}
	if count > 1 {
		return "", ErrAuditAmbiguous
	}

	return matchedID, nil
}

// GetAuditEntry retrieves a specific audit log by ID.
func GetAuditEntry(id string) (*models.AuditEntry, error) {
	fullID, err := ResolveAuditID(id)
	if err != nil {
		return nil, err
	}

	query := `SELECT id, entity_type, entity_id, action, changes_summary, snapshot_json, created_at
		FROM audit_logs
		WHERE id = ?`

	row := DB.QueryRow(query, fullID)

	var entry models.AuditEntry
	var summary sql.NullString
	var snapshot sql.NullString
	var action string

	if err := row.Scan(&entry.ID, &entry.EntityType, &entry.EntityID, &action, &summary, &snapshot, &entry.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrAuditNotFound
		}
		return nil, err
	}
	entry.Action = models.AuditAction(action)
	if summary.Valid {
		entry.ChangesSummary = summary.String
	}
	if snapshot.Valid {
		entry.SnapshotJSON = snapshot.String
	}

	return &entry, nil
}

// RevertNoteToSnapshot restores a note to a previous snapshot state.
func RevertNoteToSnapshot(noteID string, snapshotJSON string) (*models.Note, error) {
	if strings.TrimSpace(snapshotJSON) == "" {
		return nil, fmt.Errorf("snapshot is empty, cannot revert")
	}

	var snapNote models.Note
	if err := json.Unmarshal([]byte(snapshotJSON), &snapNote); err != nil {
		return nil, fmt.Errorf("failed to decode snapshot JSON: %w", err)
	}

	snapNote.ID = noteID
	snapNote.UpdatedAt = time.Now()

	// Update note in database
	if err := UpdateNote(&snapNote); err != nil {
		return nil, fmt.Errorf("failed restoring note: %w", err)
	}

	// Record explicit restored action
	_ = RecordAudit(nil, "note", noteID, models.ActionRestored, fmt.Sprintf("reverted to snapshot %s", snapNote.UpdatedAt.Format("2006-01-02 15:04")), snapNote)

	return &snapNote, nil
}
